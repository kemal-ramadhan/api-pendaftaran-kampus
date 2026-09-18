# PMB Backend Verification Scripts

Direktori ini memuat skrip otomatisasi dan verifikasi end-to-end (E2E) sistem PMB Backend.

## 1. Skrip Pengujian End-to-End (`e2e.sh`)

Skrip `scripts/e2e.sh` menjalankan alur bisnis calon mahasiswa dan otorisasi admin secara menyeluruh **melalui API Gateway** (`http://localhost:8000/api/v1/auth/...`).

### Skenario yang Diuji:
1. **Pendaftaran Calon Mahasiswa** (`POST /register`) via API Gateway.
2. **Pengambilan Kode OTP** secara otomatis via Mailpit API (`GET /api/v1/messages`).
3. **Verifikasi OTP & Pembuatan Sesi** (`POST /verify`) -> menghasilkan Access Token dan Refresh Token JWT.
4. **Akses Data Profil Mandiri** (`GET /me`) dengan validasi bearer token.
5. **Rotasi Sesi / Refresh Token** (`POST /refresh`) -> menghasilkan token baru.
6. **Pencabutan Sesi / Logout** (`POST /logout`) -> invalidasi token di Redis denylist.
7. **Pencegahan Akses Token Revoked** (`GET /me` dengan token lama) -> dipastikan **401 Unauthorized**.
8. **Login Superadmin** (`POST /login`) -> autentikasi akun superadmin.
9. **Superadmin Membuat User Internal** (`POST /admin/users`) -> membuat user dengan role `keuangan`.
10. **Login Staf Keuangan** (`POST /login`) -> autentikasi akun staf keuangan.
11. **Pengujian Otorisasi RBAC Gateway** (`GET /admin/users` oleh keuangan) -> dipastikan **403 Forbidden**.

### Cara Menjalankan

#### A. Menggunakan Docker Compose (Direkomendasikan di Server/CI)
```bash
# 1. Jalankan seluruh container
docker compose -f deploy/docker-compose.yml up -d --build

# 2. Jalankan skrip verifikasi
bash scripts/e2e.sh
```

#### B. Menggunakan Standalone / Local Dev Environment (Tanpa Docker)
Jika Docker belum terpasang di host pengembang, lingkungan dapat dijalankan secara lokal:
```bash
# Terminal 1: Jalankan Infrastruktur (PostgreSQL 5432, Redis 6379, Mailpit 1025/8025)
cd scripts/infra
go run .

# Terminal 2: Jalankan Auth Service (Port 8080)
cd services/auth-service
go run ./cmd/main.go

# Terminal 3: Jalankan API Gateway (Port 8000)
cd services/api-gateway
go run ./cmd/main.go

# Terminal 4: Jalankan Skrip E2E
bash scripts/e2e.sh
```
