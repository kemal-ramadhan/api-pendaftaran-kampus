# Aturan Kerja AI Agent — Sistem PMB

Dokumen ini adalah pedoman dan aturan kerja wajib (*mandatory rules*) bagi seluruh AI Agent yang mengerjakan Sistem Pendaftaran Mahasiswa Baru (PMB) di repository ini. Aturan ini bersumber dari panduan instruksi utama proyek dan tidak boleh dilanggar dalam kondisi apa pun.

---

## 1. Sumber Kebenaran
1. **`PRD-LENGKAP.md`** di root repository adalah **SATU-SATUNYA** spesifikasi resmi.
   - Bagian A: Status & masalah terbuka.
   - Bagian B: PRD Fase 1 (Auth Service & API Gateway — sudah terverifikasi E2E 43/43).
   - Bagian C: PRD Fase 2 (Seluruh service PMB + OCR KTP).
   - Bagian D: Change Request (CR-001: Payment Gateway Duitku).
   - Bagian E: Keputusan teknis tambahan & pemetaan tahap.
2. Jika Bagian D atau E bertentangan dengan Bagian B atau C, maka **Bagian D / E yang berlaku** (karena lebih baru).
3. Dokumen pendukung: `docs/DECISIONS.md`, `docs/API.md`, `docs/CHANGE_REQUESTS.md`, `docs/schema-dan-class-diagram.md`, `docs/flowchart-sistem-pmb.pdf`, `docs/use-case-sistem-pmb.pdf`.
4. Untuk integrasi Duitku, wajib membuka dokumentasi resmi di `https://docs.duitku.com/pop/en` sebelum implementasi dan dicocokkan dengan `PRD-LENGKAP.md` Bagian D.1.

---

## 2. Fakta Penting Repository
- **Auth Service & API Gateway** sudah TERVERIFIKASI (E2E 43/43 lulus di Docker dari DB kosong per 19 September 2026).
- Service fase 2 (`organization`, `identity`, `notification`, `admission`, `finance`, `exam`, `scoring`, `selection`, `student`, `nim`, `ocr`) sudah memiliki draf kode dari agent sebelumnya, tetapi **BELUM TERVERIFIKASI**.
- **JANGAN** mempercayai klaim "selesai" dari laporan/komentar sebelumnya tanpa verifikasi mandiri. Anggap seluruh kode fase 2 sebagai draf yang wajib diaudit terhadap PRD.
- Lingkungan Pengembang: **Windows + PowerShell + Docker Desktop**. Script bash dijalankan lewat Git Bash:
  `& "C:\Program Files\Git\bin\bash.exe" scripts/e2e.sh`.
  Semua instruksi yang diberikan kepada pengguna harus dapat dijalankan di PowerShell (atau Git Bash untuk file `.sh`).

---

## 3. Aturan Kerja & Keamanan (Wajib Dipatuhi)

### A. Keamanan & Batasan Integritas
1. **Jangan mengubah perilaku Auth Service & API Gateway** yang sudah lulus, kecuali untuk integrasi yang eksplisit diminta PRD (mis. Bagian C.4.10 validasi scope prodi, Duitku route di Gateway, dll.) atau perbaikan bug kritis. Setiap perubahan wajib tetap meluluskan seluruh E2E Fase 1.
2. **Jangan pernah commit rahasia/secret**: file `.env`, `*.pem`, API key Duitku, folder `certs/`, `reference/`, atau `postgres-data/`.
3. **Jangan mengurangi keamanan demi kecepatan**:
   - Signature HMAC API Gateway wajib divalidasi di setiap service (`pkg/gatewayauth`).
   - Otorisasi role & scope wajib ditegakkan di level service/controller.
   - Pendaftar hanya boleh mengakses data miliknya sendiri (`X-User-ID`).
   - Validasi input ketat (mimes, ukuran file, schema DTO).
   - Nilai uang wajib integer (dalam Rupiah, tanpa float).
   - Webhook pembayaran wajib idempoten dan memverifikasi HMAC constant-time.
   - Penomoran NIM harus unik dan aman terhadap konkurensi.
   - NIK tersimpan terenkripsi (AES-256-GCM) dan tidak boleh bocor di log/response.
   - Presigned URL dokumen dengan masa berlaku terbatas (5 menit untuk admin).
4. **Jangan melemahkan atau menghapus test** demi membuat build lulus. Jika test gagal karena kode salah, perbaiki kodenya. Jika test keliru terhadap PRD, jelaskan terlebih dahulu di laporan sebelum diubah.
5. **Dilarang force push** (`--force`), dilarang rewrite history, dan dilarang menghapus branch.

### B. Standar Kode
- Pola arsitektur setiap service Go seragam dengan `auth-service`:
  `cmd/main.go`, `internal/` (`config`, `model`, `repository`, `service`, `controller`, `dto`, `middleware`, `route`).
- Database & migrasi: Menggunakan migrasi SQL murni (`golang-migrate`), bukan AutoMigrate GORM.
- Satu versi Go untuk seluruh `go.mod` dan `Dockerfile` (seragam dengan base image Go yang stabil).
- Setiap module Go wajib `go mod tidy` bersih dan file `go.sum` di-commit. CI/pemeriksaan lokal gagal bila `go mod tidy` mengubah file.
- Format response standar: menggunakan package `pkg/response`. Pesan ke pengguna (`message`) dalam **Bahasa Indonesia**; identifier, nama tabel, kolom, dan field JSON dalam **Bahasa Inggris**.
- Kemandirian Service: Service dilarang membaca database service lain. Komunikasi sinkron antar-service lewat `/internal/*` + header `X-Internal-Token`. Komunikasi asinkron perubahan status menggunakan Transactional Outbox Pattern.
- Kontainerisasi: Dockerfile multi-stage, BuildKit cache mount (`/go/pkg/mod` dan `/root/.cache/go-build`), non-root user, build context dari root repository.
- GORM logger: Tidak mencetak `ErrRecordNotFound` dan tidak mencetak SQL mentah kecuali `LOG_LEVEL=debug`.

---

## 4. Definisi "Selesai" (Definition of Done per Tahap)
Satu tahap hanya dapat dinyatakan selesai jika **SEMUA** poin berikut terpenuhi:
1. **Module Go / Python**:
   - Di setiap module: `go mod tidy` tidak mengubah apa pun (`git diff --exit-code go.mod go.sum`).
   - `go build ./...`, `go vet ./...`, `go test ./...` lulus.
   - Pada `ocr-service`: `pytest` lulus dan linting dasar bersih.
2. **Kontainerisasi & Orkestrasi**:
   - `docker compose -f deploy/docker-compose.yml down -v`
   - `docker compose -f deploy/docker-compose.yml up -d --build`
   - **SEMUA** container berstatus **healthy** (dibuktikan dengan output `docker compose ps`).
3. **Pengujian End-to-End**:
   - `scripts/e2e.sh` dijalankan SETELAH semua container berstatus healthy.
   - Hasil pengujian: **FAIL: 0**.
   - Salin dan tempel ringkasan output (total PASS/FAIL dan baris FAIL jika ada).
   - E2E yang dijalankan saat build gagal atau container tidak healthy **TIDAK DIHITUNG**.
4. **Dokumentasi Lengkap**:
   - Swagger up-to-date di setiap service.
   - `docs/API.md` memuat endpoint, contoh request/response, dan error code.
   - `docs/PMB.postman_collection.json` diperbarui per folder service.
   - `docs/class-diagram-<service>.md` dan `docs/erd-<service>.md` (Mermaid dari kode aktual).
   - `docs/DECISIONS.md` untuk keputusan baru.
   - `docs/AUDIT-REPORT.md` dan tabel keterlacakan use case diperbarui.
5. **Commit & Push**:
   - Mematuhi prosedur commit & push pada Bagian 5.
   - Jika Docker tidak dapat dijalankan di lingkungan agent, wajib menyatakan secara eksplisit: **"BELUM TERVERIFIKASI DI DOCKER"** dan memberikan perintah persis untuk pengguna. Dilarang mengklaim "lulus" tanpa eksekusi langsung.

---

## 5. Prosedur Commit & Push (Wajib Dijalankan Setiap Push)
1. Tarik perubahan terbaru:
   ```bash
   git pull --rebase origin main
   ```
2. Pemeriksaan Secret (kedua perintah harus menghasilkan output **KOSONG**):
   ```bash
   git ls-files | grep -E '(^|/)\.env$|\.pem$|certs/|postgres-data|reference/'
   git diff --cached | grep -iE 'DUITKU_API_KEY=.+|BEGIN (RSA )?PRIVATE KEY'
   ```
3. Periksa status working tree:
   ```bash
   git status
   ```
   (Hanya boleh memuat perubahan yang disengaja).
4. Buat commit menggunakan format **Conventional Commits**:
   - `feat(<service>): ...`
   - `fix(<service>): ...`
   - `docs: ...`
   - `test(<service>): ...`
   - `chore: ...`
   (Satu commit per unit kerja logis).
5. Push ke branch utama:
   ```bash
   git push origin main
   ```
   (Jika ditolak non-fast-forward, ulangi langkah 1. **Dilarang keras memakai `--force`**).
6. Verifikasi hasil push:
   ```bash
   git log origin/main --oneline -5
   ```
   Sertakan hash commit di setiap laporan.

---

## 6. Protokol Format Laporan di Setiap Checkpoint
Di setiap checkpoint atau akhir tahap, AI Agent wajib menyajikan laporan dengan format terstandarisasi:

```markdown
## Laporan <Langkah / Tahap>
1. Ringkasan (3–5 kalimat, Bahasa Indonesia sederhana)
2. Hasil Pemeriksaan:
   - go mod tidy / build / vet / test per module: OK / GAGAL
   - docker compose ps: daftar container + status
   - e2e: Total PASS X, FAIL Y (+ baris FAIL bila ada)
   *(Jika tidak dapat dijalankan: tulis "BELUM TERVERIFIKASI DI DOCKER")*
3. Perubahan: file/service yang diubah, endpoint baru/berubah
4. Temuan & keputusan baru (juga dicatat di DECISIONS.md)
5. Commit & Push: hash commit, hasil git log origin/main --oneline -3
6. Perintah Verifikasi Manual untuk Pengguna (PowerShell, maksimal 10 langkah)
7. Risiko & sisa pekerjaan
```
Lalu **BERHENTI** dan menunggu respon pengguna sebelum melanjutkan.
