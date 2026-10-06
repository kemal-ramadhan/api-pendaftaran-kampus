package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
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

	// 3. Mock Mailpit (SMTP :1025, HTTP :8025) - hanya jika belum ada Mailpit berjalan
	connCheck, errConn := net.DialTimeout("tcp", "127.0.0.1:1025", 500*time.Millisecond)
	if errConn == nil {
		connCheck.Close()
		fmt.Println("📧 Mailpit resmi terdeteksi aktif di 127.0.0.1:1025 (melewati mock)")
	} else {
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

		// HTTP API Listener
		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
			msgs := store.List()
			type summary struct {
				ID      string `json:"ID"`
				Snippet string `json:"Snippet"`
			}
			var summaries []summary
			for _, m := range msgs {
				summaries = append(summaries, summary{ID: m.ID, Snippet: m.Snippet})
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
			})
		})

		httpServer := &http.Server{Addr: "127.0.0.1:8025", Handler: mux}
		go func() {
			fmt.Println("📬 Mailpit Web/API aktif di http://127.0.0.1:8025")
			_ = httpServer.ListenAndServe()
		}()
		defer httpServer.Close()
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
