package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alicebob/miniredis/v2"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type MailMessage struct {
	ID      string `json:"ID"`
	Snippet string `json:"Snippet"`
	Body    string `json:"Body"`
	To      string `json:"To"`
}

type MailStore struct {
	mu       sync.RWMutex
	messages []MailMessage
}

func (s *MailStore) Add(msg MailMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append([]MailMessage{msg}, s.messages...)
}

func (s *MailStore) List() []MailMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]MailMessage, len(s.messages))
	copy(res, s.messages)
	return res
}

func (s *MailStore) Get(id string) (MailMessage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.messages {
		if m.ID == id {
			return m, true
		}
	}
	return MailMessage{}, false
}

func main() {
	fmt.Println("🚀 Menjalankan Shared Local Infrastructure (Postgres, Redis, Mailpit)...")

	// 1. Miniredis di port 6379
	sRedis := miniredis.NewMiniRedis()
	if err := sRedis.StartAddr("127.0.0.1:6379"); err != nil {
		fmt.Printf("Peringatan Redis 6379: %v (mungkin sudah berjalan)\n", err)
	} else {
		defer sRedis.Close()
		fmt.Println("🔴 Redis aktif di 127.0.0.1:6379")

		// Jalankan ticker background agar waktu virtual miniredis bergerak secara real-time
		// sehingga kunci TTL (rate limit, OTP, session) dapat kedaluwarsa secara otomatis
		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				sRedis.FastForward(1 * time.Second)
			}
		}()
	}

	// 2. Embedded PostgreSQL di port 5432
	homeDir, _ := os.UserHomeDir()
	stalePid := homeDir + "/.embedded-postgres-go/extracted/data/postmaster.pid"
	_ = os.Remove(stalePid)

	pgCfg := embeddedpostgres.DefaultConfig().
		Port(5432).
		Database("pmb_auth").
		Username("pmb_user").
		Password("pmb_password")
	pg := embeddedpostgres.NewDatabase(pgCfg)
	if err := pg.Start(); err != nil {
		fmt.Printf("Peringatan Postgres 5432: %v (mungkin sudah berjalan)\n", err)
	} else {
		defer pg.Stop()
		fmt.Println("🐘 PostgreSQL aktif di 127.0.0.1:5432 (DB: pmb_auth, User: pmb_user)")
		initOtherDBs("postgres://pmb_user:pmb_password@127.0.0.1:5432/pmb_auth?sslmode=disable")
	}

	// 3. Mailpit (SMTP :1025, Web UI :8025)
	connCheck, errConn := net.DialTimeout("tcp", "127.0.0.1:1025", 500*time.Millisecond)
	if errConn == nil {
		connCheck.Close()
		fmt.Println("📧 Mailpit resmi terdeteksi aktif di 127.0.0.1:1025 (melewati inisialisasi)")
	} else {
		var mailpitCmd *exec.Cmd
		mailpitPath := findMailpit()
		if mailpitPath != "" {
			fmt.Printf("📬 Menjalankan Mailpit resmi (%s) di :8025 (UI) & :1025 (SMTP)...\n", mailpitPath)
			mailpitCmd = exec.Command(mailpitPath, "-l", "127.0.0.1:8025", "-s", "127.0.0.1:1025", "--disable-version-check")
			mailpitCmd.Stdout = os.Stdout
			mailpitCmd.Stderr = os.Stderr
			if err := mailpitCmd.Start(); err != nil {
				fmt.Printf("Peringatan Mailpit resmi: %v. Menggunakan mock internal...\n", err)
				mailpitCmd = nil
			} else {
				defer func() {
					if mailpitCmd != nil && mailpitCmd.Process != nil {
						_ = mailpitCmd.Process.Kill()
					}
				}()
			}
		}

		if mailpitCmd == nil {
			store := &MailStore{}

			// SMTP Listener
			smtpListener, err := net.Listen("tcp", "127.0.0.1:1025")
			if err != nil {
				fmt.Printf("Peringatan SMTP 1025: %v\n", err)
			} else {
				defer smtpListener.Close()
				fmt.Println("📧 Mailpit SMTP aktif di 127.0.0.1:1025")
				go func() {
					for {
						conn, err := smtpListener.Accept()
						if err != nil {
							return
						}
						go handleSMTP(conn, store)
					}
				}()
			}

			// HTTP API & Web UI Listener
			mux := http.NewServeMux()

			// Root Web UI (Mencegah 404 saat dibuka di browser)
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/view") {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write([]byte(mailpitWebUIHtml))
			})

			mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
				msgs := store.List()
				type summary struct {
					ID      string `json:"ID"`
					Snippet string `json:"Snippet"`
					To      string `json:"To"`
				}
				var summaries []summary
				for _, m := range msgs {
					summaries = append(summaries, summary{ID: m.ID, Snippet: m.Snippet, To: m.To})
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"total":    len(msgs),
					"messages": summaries,
				})
			})

			mux.HandleFunc("/api/v1/message/", func(w http.ResponseWriter, r *http.Request) {
				parts := strings.Split(r.URL.Path, "/")
				id := parts[len(parts)-1]
				msg, ok := store.Get(id)
				if !ok {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"ID":      msg.ID,
					"Snippet": msg.Snippet,
					"Text":    msg.Body,
					"HTML":    msg.Body,
					"To":      msg.To,
				})
			})

			httpServer := &http.Server{Addr: "127.0.0.1:8025", Handler: mux}
			go func() {
				fmt.Println("📬 Mailpit Web/API aktif di http://127.0.0.1:8025")
				_ = httpServer.ListenAndServe()
			}()
			defer httpServer.Close()
		}
	}

	fmt.Println("Infrastruktur siap! Menunggu shutdown signal (SIGINT/SIGTERM)...")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("Menutup infrastruktur...")
}

func handleSMTP(conn net.Conn, store *MailStore) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	writer.WriteString("220 mailpit.local ESMTP MockMailpit\r\n")
	writer.Flush()

	var toAddr string
	var inData bool
	var bodyBuilder strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		lineTrim := strings.TrimRight(line, "\r\n")

		if inData {
			if lineTrim == "." {
				inData = false
				body := bodyBuilder.String()
				msgID := uuid.New().String()

				clean := strings.ReplaceAll(body, "\r", " ")
				clean = strings.ReplaceAll(clean, "\n", " ")
				snippet := clean
				if len(snippet) > 150 {
					snippet = snippet[:150]
				}

				store.Add(MailMessage{
					ID:      msgID,
					Snippet: snippet,
					Body:    body,
					To:      toAddr,
				})

				fmt.Fprintf(writer, "250 OK: queued as %s\r\n", msgID)
				writer.Flush()
				continue
			}
			bodyBuilder.WriteString(line)
			continue
		}

		upper := strings.ToUpper(lineTrim)
		if strings.HasPrefix(upper, "EHLO") || strings.HasPrefix(upper, "HELO") {
			writer.WriteString("250-mailpit.local\r\n250-PIPELINING\r\n250-8BITMIME\r\n250 OK\r\n")
			writer.Flush()
		} else if strings.HasPrefix(upper, "MAIL FROM:") {
			writer.WriteString("250 OK\r\n")
			writer.Flush()
		} else if strings.HasPrefix(upper, "RCPT TO:") {
			re := regexp.MustCompile(`<([^>]+)>`)
			matches := re.FindStringSubmatch(lineTrim)
			if len(matches) > 1 {
				toAddr = matches[1]
			}
			writer.WriteString("250 OK\r\n")
			writer.Flush()
		} else if strings.HasPrefix(upper, "DATA") {
			inData = true
			bodyBuilder.Reset()
			writer.WriteString("354 Start mail input; end with <CRLF>.<CRLF>\r\n")
			writer.Flush()
		} else if strings.HasPrefix(upper, "QUIT") {
			writer.WriteString("221 Bye\r\n")
			writer.Flush()
			break
		} else if strings.HasPrefix(upper, "RSET") {
			bodyBuilder.Reset()
			writer.WriteString("250 OK\r\n")
			writer.Flush()
		} else if strings.HasPrefix(upper, "NOOP") {
			writer.WriteString("250 OK\r\n")
			writer.Flush()
		} else {
			writer.WriteString("250 OK\r\n")
			writer.Flush()
		}
	}
}

func initOtherDBs(connStr string) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("Gagal koneksi ke postgres untuk inisialisasi DB: %v\n", err)
		return
	}
	defer db.Close()

	dbs := []string{
		"pmb_organization",
		"pmb_identity",
		"pmb_notification",
		"pmb_admission",
		"pmb_finance",
		"pmb_exam",
		"pmb_scoring",
		"pmb_selection",
		"pmb_student",
		"pmb_nim",
	}

	for _, name := range dbs {
		var exists bool
		_ = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists)
		if !exists {
			_, err = db.Exec("CREATE DATABASE " + name)
			if err != nil {
				fmt.Printf("Gagal membuat DB %s: %v\n", name, err)
			} else {
				fmt.Printf("📦 Database %s berhasil dibuat\n", name)
			}
		}
	}
}

func findMailpit() string {
	candidates := []string{
		"C:\\Users\\inboo\\go\\bin\\mailpit.exe",
		filepath.Join("..", "..", "bin", "mailpit.exe"),
		filepath.Join("bin", "mailpit.exe"),
		"mailpit.exe",
		"mailpit",
	}
	for _, c := range candidates {
		if path, err := exec.LookPath(c); err == nil {
			return path
		}
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	return ""
}

const mailpitWebUIHtml = `<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Mailpit - Local Email Testing</title>
  <link rel="icon" href="data:image/svg+xml,<svg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 100 100%22><text y=%22.9em%22 font-size=%2290%22>📬</text></svg>">
  <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-slate-100 text-slate-800 min-h-screen flex flex-col font-sans">
  <header class="bg-slate-900 text-white px-6 py-3 flex items-center justify-between shadow-md">
    <div class="flex items-center gap-3">
      <span class="text-2xl">📬</span>
      <div>
        <h1 class="font-bold text-lg leading-tight flex items-center gap-2">
          <span>Mailpit</span>
          <span class="text-[10px] bg-emerald-500 text-white px-2 py-0.5 rounded-full font-mono uppercase tracking-wider">PMB Local</span>
        </h1>
        <p class="text-xs text-slate-400">SMTP: 127.0.0.1:1025 | Web UI: 127.0.0.1:8025</p>
      </div>
    </div>
    <div class="flex items-center gap-3 text-xs">
      <button onclick="fetchMessages()" class="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 rounded-lg border border-slate-700 flex items-center gap-1.5 transition cursor-pointer">
        <span>🔄 Refresh Inbox</span>
      </button>
      <span id="unreadBadge" class="bg-blue-600 text-white px-2.5 py-1 rounded-full font-bold">0 Pesan</span>
    </div>
  </header>

  <div class="flex-1 max-w-7xl w-full mx-auto p-4 sm:p-6 grid grid-cols-1 md:grid-cols-12 gap-6">
    <div class="md:col-span-5 bg-white rounded-xl border border-slate-200 shadow-sm flex flex-col overflow-hidden">
      <div class="p-3 bg-slate-50 border-b border-slate-200 flex items-center justify-between">
        <span class="font-bold text-xs text-slate-700 uppercase tracking-wider">Inbox Email Masuk</span>
        <span class="text-xs text-slate-500" id="statusText">Memuat...</span>
      </div>
      <div id="messageList" class="flex-1 overflow-y-auto divide-y divide-slate-100 max-h-[75vh]"></div>
    </div>

    <div class="md:col-span-7 bg-white rounded-xl border border-slate-200 shadow-sm flex flex-col overflow-hidden">
      <div id="emptyDetail" class="p-12 text-center text-slate-400 my-auto">
        <span class="text-4xl block mb-2">✉️</span>
        <p class="text-sm font-medium">Pilih salah satu email di sebelah kiri untuk melihat isi pesan & kode OTP.</p>
      </div>
      <div id="detailPanel" class="hidden flex-1 flex flex-col">
        <div class="p-4 bg-slate-50 border-b border-slate-200 space-y-2">
          <div class="flex items-center justify-between">
            <span class="text-xs text-slate-500 font-mono" id="msgIdText"></span>
            <div id="otpBadgeContainer"></div>
          </div>
          <h2 class="text-base font-bold text-slate-900" id="msgSubjectText">Subjek Email</h2>
          <div class="text-xs text-slate-600 space-y-0.5">
            <div><strong class="text-slate-500">Penerima:</strong> <span id="msgToText" class="font-mono"></span></div>
          </div>
        </div>
        <div class="p-4 flex-1 overflow-y-auto">
          <div id="msgBodyContent" class="text-xs font-mono bg-slate-50 p-4 rounded-lg border border-slate-200 whitespace-pre-wrap leading-relaxed"></div>
        </div>
      </div>
    </div>
  </div>

  <footer class="bg-white border-t border-slate-200 py-3 text-center text-xs text-slate-500">
    Mailpit Simulator &bull; Universitas Bakti Tunas Husada PMB System
  </footer>

  <script>
    let activeId = null;
    let allMessages = [];

    async function fetchMessages() {
      try {
        const res = await fetch('/api/v1/messages');
        const data = await res.json();
        allMessages = data.messages || [];
        document.getElementById('unreadBadge').innerText = allMessages.length + ' Pesan';
        document.getElementById('statusText').innerText = 'Sinkron (' + new Date().toLocaleTimeString() + ')';
        renderList();
      } catch (e) {
        document.getElementById('statusText').innerText = 'Gagal memuat API';
      }
    }

    function renderList() {
      const container = document.getElementById('messageList');
      if (allMessages.length === 0) {
        container.innerHTML = '<div class="p-8 text-center text-slate-400 text-xs">Belum ada email yang ditangkap di Mailpit.</div>';
        return;
      }
      container.innerHTML = allMessages.map(m => {
        const activeClass = m.ID === activeId ? 'bg-blue-50 border-l-4 border-blue-600' : 'hover:bg-slate-50';
        const otpMatch = m.Snippet.match(/\\b(\\d{6})\\b/);
        const otpBadge = otpMatch ? '<span class=\"bg-emerald-100 text-emerald-800 font-bold px-2 py-0.5 rounded text-[10px]\">OTP: ' + otpMatch[1] + '</span>' : '';
        return '<div onclick=\"loadMessage(\\'' + m.ID + '\\')\" class=\"p-3.5 cursor-pointer transition ' + activeClass + '\">' +
          '<div class=\"flex items-center justify-between mb-1\">' +
            '<span class=\"text-[11px] font-mono text-slate-500 truncate max-w-[200px]\">' + (m.To || 'calon.maba') + '</span>' +
            otpBadge +
          '</div>' +
          '<div class=\"text-xs font-semibold text-slate-800 truncate\">' + (m.Snippet.substring(0, 60)) + '</div>' +
          '<div class=\"text-[11px] text-slate-500 truncate mt-0.5\">' + (m.Snippet.substring(60, 140)) + '</div>' +
        '</div>';
      }).join('');
    }

    async function loadMessage(id) {
      activeId = id;
      renderList();
      try {
        const res = await fetch('/api/v1/message/' + id);
        const msg = await res.json();
        document.getElementById('emptyDetail').classList.add('hidden');
        document.getElementById('detailPanel').classList.remove('hidden');
        document.getElementById('msgIdText').innerText = 'ID: ' + msg.ID;
        document.getElementById('msgBodyContent').innerText = msg.Text || msg.Snippet;
        
        const toMatch = (msg.Text || '').match(/To:\\s*([^\\r\\n]+)/i);
        document.getElementById('msgToText').innerText = toMatch ? toMatch[1].trim() : (msg.To || '-');
        
        const subjMatch = (msg.Text || '').match(/Subject:\\s*([^\\r\\n]+)/i);
        document.getElementById('msgSubjectText').innerText = subjMatch ? subjMatch[1].trim() : 'Email Pemberitahuan PMB';

        const otpMatch = (msg.Text || '').match(/\\b(\\d{6})\\b/);
        const badgeCont = document.getElementById('otpBadgeContainer');
        if (otpMatch) {
          badgeCont.innerHTML = '<div class=\"flex items-center gap-2\"><span class=\"bg-emerald-100 text-emerald-800 border border-emerald-300 font-mono font-bold text-xs px-2.5 py-1 rounded-md\">OTP: ' + otpMatch[1] + '</span><button onclick=\"navigator.clipboard.writeText(\\'' + otpMatch[1] + '\\'); alert(\\'OTP ' + otpMatch[1] + ' disalin!\\')\" class=\"text-xs bg-slate-800 hover:bg-slate-700 text-white px-2.5 py-1 rounded transition cursor-pointer\">Salin OTP</button></div>';
        } else {
          badgeCont.innerHTML = '';
        }
      } catch (e) {
        alert('Gagal membuka detail pesan: ' + e);
      }
    }

    fetchMessages();
    setInterval(fetchMessages, 3000);
  </script>
</body>
</html>`

