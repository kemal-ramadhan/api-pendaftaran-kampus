# Aturan Kerja AI Agent — Sistem PMB (Fase 2)

Dokumen ini adalah pedoman dan aturan kerja wajib (*mandatory rules*) bagi AI Agent selama pelaksanaan implementasi seluruh tahapan PMB Fase 2 berdasarkan acuan `PRD-fase-2.md` dan `PROMPT-per-tahap.md`. Aturan ini harus dipatuhi tanpa pengecualian di setiap tahap pengembangan.

---

## 1. Prinsip Integritas & Backward Compatibility
1. **Perilaku Layanan Fase 1**: Jangan pernah mengubah perilaku **Auth Service** dan **API Gateway** yang sudah lulus pengujian E2E (43/43 skenario), kecuali:
   - Penambahan yang secara eksplisit diminta pada `PRD-fase-2.md` Bagian 4.10 (misalnya: validasi scope prodi ke Organization Service saat superadmin mengatur role).
   - Perbaikan bug kritis (*bugfix*).
2. **Kemandirian Database Service**: Satu service memiliki basis data sendiri di satu instance PostgreSQL (`pmb_organization`, `pmb_identity`, dst.). Tidak ada service yang diperbolehkan membaca langsung tabel atau database milik service lain.
3. **ID Global Pengguna**: UUID `users.id` dari Auth Service adalah identitas global tunggal pengguna di seluruh ekosistem kampus.

---

## 2. Standar Pengembangan Microservice Baru
Setiap service baru wajib mengikuti konvensi arsitektur yang seragam dengan `auth-service`:
1. **Struktur Folder & Kode**:
   - Pola modular: `cmd/main.go`, `internal/config`, `internal/model`, `internal/repository`, `internal/service`, `internal/controller`, `internal/route`, `internal/dto`.
   - Menggunakan Go 1.25, Gin, GORM, dan library standar yang sudah disepakati.
2. **Keamanan & Autentikasi**:
   - Wajib menggunakan middleware `pkg/gatewayauth` untuk memverifikasi HMAC Signature API Gateway pada seluruh endpoint publik/terproteksi klien (kecuali `/health*` dan `/internal/*`).
   - Ekstraksi identitas user wajib membaca header `X-User-ID`, `X-User-Roles`, dan `X-User-Scopes` dari `pkg/gatewayauth`.
   - Otorisasi detail berbasis role dan scope wajib diimplementasikan di level controller/service.
   - Endpoint antar-layanan `/internal/*` wajib dilindungi oleh `pkg/internalauth` dengan header `X-Internal-Token` dan tidak diekspos melalui API Gateway.
3. **Database & Migrasi**:
   - Menyertakan file migrasi SQL bersih di direktori `migrations/`.
   - Tabel dan kolom teknis menyertakan: `created_at`, `updated_at`, `deleted_at`, serta `created_by` (jika relevan).
4. **Dokumentasi API (Swagger)**:
   - Dilengkapi anotasi OpenAPI/Swagger (swaggo) di setiap handler controller.
   - Menghasilkan file swagger yang valid di direktori `docs/` tiap service.
5. **Kontainerisasi & Konfigurasi**:
   - `Dockerfile` multi-stage build dengan optimasi BuildKit cache mount (`/go/pkg/mod` dan `/root/.cache/go-build`).
   - Non-root user pada runtime container.
   - File konfigurasi `.env.example` dengan placeholder lengkap dan dokumentatif.
   - Endpoint health check: `/health` (liveness) dan `/health/ready` (readiness).
6. **Integrasi Gateway**:
   - Menambahkan definisi upstream service dan routing deklaratif di `services/api-gateway/config/routes.yaml`.

---

## 3. Protokol Verifikasi Kualitas Sebelum Melapor
Sebelum menandai suatu tahap selesai atau membuat laporan:
1. **Pemeriksaan Kode Go**:
   - `go build ./...`
   - `go vet ./...`
   - `go test -v ./...` (unit test lulus dengan coverage target service ≥ 70%)
2. **Pemeriksaan Kontainer & Infrastruktur**:
   - `docker compose -f deploy/docker-compose.yml down -v`
   - `docker compose -f deploy/docker-compose.yml up -d --build`
   - Pastikan seluruh container berstatus **healthy** (termasuk Postgres, Redis, Mailpit, MinIO, Auth, Gateway, dan service yang aktif).
3. **Pengujian End-to-End**:
   - Jalankan `bash scripts/e2e.sh`.
   - Seluruh skenario pengujian wajib **PASS** (0 FAIL), mencakup skenario lama (Fase 1) dan skenario baru yang ditambahkan di tahap tersebut.

---

## 4. Standar Dokumentasi per Tahap
Setiap milestone/tahap wajib memperbarui dan menyertakan dokumen:
1. **`docs/API.md`**: Mencatat endpoint baru, payload request/response, error code baru, serta contoh cURL.
2. **Postman Collection**: Menambahkan/memperbarui folder request terkait di `docs/PMB.postman_collection.json`.
3. **Diagram Kelas (`docs/class-diagram-<service>.md`)**: Diagram UML Mermaid yang diturunkan dari kode aktual (model, service, repository, controller).
4. **Diagram Relasi Entitas (`docs/erd-<service>.md`)**: Diagram ERD Mermaid skema database service terkait.
5. **`docs/DECISIONS.md`**: Mencatat keputusan arsitektur baru atau kompromi teknis yang diambil.
6. **Status PRD**: Memperbarui status use case di `PRD-fase-2.md` Bagian 3 (Keterlacakan Use Case).

---

## 5. Manajemen Versi (Git & Commit)
1. Commit dilakukan secara terstruktur per milestone dengan format Conventional Commits (misalnya: `feat(org): implement faculty and study program endpoints`, `test(org): add unit and e2e tests`).
2. Jangan pernah melakukan commit untuk file secret (`.env`, private key `.pem`, kredensial produksi).

---

## 6. Protokol Pelaporan AI Agent
Setelah menyelesaikan suatu tahap, AI Agent wajib menyajikan laporan dengan format:
1. **Hasil Pengujian**:
   - Status `go build`, `go vet`, `go test`.
   - Status container Docker Compose.
   - Ringkasan eksekusi `scripts/e2e.sh` (Jumlah PASS / FAIL).
2. **Daftar Endpoint Baru**: Metode HTTP, path, hak akses role/scope, serta deskripsi singkat.
3. **Keputusan Arsitektur Baru**: Penjelasan keputusan teknis yang dicatat di `docs/DECISIONS.md`.
4. **Instruksi Verifikasi Manual**: Perintah cURL atau langkah konkret yang dapat diuji langsung oleh user/lead.
5. **BERHENTI**: Menunggu konfirmasi dan instruksi manual checklist dari user sebelum melangkah ke tahap berikutnya.

---

## 7. Manajemen Perubahan (Change Request)
1. Setiap instruksi perubahan atau arahan baru dari lead (Pak Kemal) wajib dicatat terlebih dahulu di `docs/CHANGE_REQUESTS.md`.
2. Lakukan analisis dampak (service, tabel, endpoint, test, tahap terdampak) dan usulkan pembaruan PRD sebelum memodifikasi kode.
