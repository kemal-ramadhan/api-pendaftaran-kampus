# PMB Backend Verification Scripts

Direktori ini memuat skrip otomatisasi dan verifikasi end-to-end (E2E) sistem PMB Backend.

## 1. Skrip Pengujian End-to-End (`e2e.sh`)

Skrip `scripts/e2e.sh` menjalankan alur bisnis calon mahasiswa, otorisasi admin, serta skenario pengujian negatif dan keamanan secara menyeluruh **melalui API Gateway** (`http://localhost:8000/api/v1/auth/...`).

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
12. **Skenario Negatif Keamanan**:
    - Percobaan OTP salah berulang hingga terkunci (`OTP_LOCKED`, HTTP 429).
    - Login salah 5x berturut-turut hingga lockout (`TOO_MANY_ATTEMPTS`, HTTP 429).
    - Login email tak terdaftar (`INVALID_CREDENTIALS`, HTTP 401, pesan sama persis dengan password salah).
    - Token Reuse Detection: Refresh token lama yang sudah dirotasi digunakan kembali (`TOKEN_REVOKED`, HTTP 401).
    - Penolakan Refresh Token sebagai Access Token (`TOKEN_INVALID`, HTTP 401).
    - Anti-Spoofing: Injeksi header palsu `X-User-ID` dari klien dibersihkan/ditimpa.
    - Pemblokiran Akses Gateway ke Endpoint Internal Hilir (`/internal/*` -> HTTP 404).
    - Alur Lupa & Reset Password (`POST /forgot-password` -> baca token dari Mailpit -> `POST /reset-password` -> login sukses dengan password baru).

---

## 2. Cara Menjalankan

### A. Menggunakan Docker Asli (Wajib untuk Verifikasi Resmi & CI/CD)
> [!IMPORTANT]
> Pengujian resmi sistem HARUS dijalankan terhadap container Docker asli (PostgreSQL 15, Redis 7, dan Mailpit resmi).

```bash
# 1. Jalankan seluruh container
docker compose -f deploy/docker-compose.yml up -d --build

# 2. Jalankan skrip verifikasi (Linux / macOS)
bash scripts/e2e.sh

# 2. Jalankan di Windows (menggunakan Git Bash)
"C:\Program Files\Git\bin\bash.exe" scripts/e2e.sh
```

---

### B. Lingkungan Standalone / In-Memory (`scripts/infra`)
> [!WARNING]
> **LABEL PERINGATAN (PENTING):**
> `scripts/infra/` adalah infrastruktur tiruan/mock berbasis in-memory (`miniredis`, `embedded-postgres`, dan listener SMTP tiruan).
> **Hanya digunakan untuk keperluan development darurat ketika Docker engine tidak terpasang di komputer pengembang.**
> Komponen ini **BUKAN PENGGANTI TES DI DOCKER ASLI**, karena perilaku concurrency, storage disk, dan persistensi state berbeda dengan PostgreSQL dan Redis produksi.

Jika terpaksa menjalankan tanpa Docker:
```bash
# Terminal 1: Jalankan Infrastruktur Tiruan (Postgres 5432, Redis 6379, Mailpit 1025/8025)
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
