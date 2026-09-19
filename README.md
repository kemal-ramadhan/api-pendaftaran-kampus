# PMB Backend — Sistem Penerimaan Mahasiswa Baru

Repositori monorepo backend Sistem PMB (Fase 1: **Auth Service** dan **API Gateway**).

---

## 1. Arsitektur & Tech Stack
- **Language**: Go 1.25+
- **HTTP Framework**: Gin
- **Database**: PostgreSQL 15, GORM, `golang-migrate`
- **Cache & Session**: Redis 7
- **Authentication**: JWT Asymmetric RS256 + JWKS endpoint (`/.well-known/jwks.json`)
- **Email Development**: Mailpit (penangkap email OTP & reset password)
- **Containerization**: Docker Compose

---

## 2. Struktur Monorepo
```
pmb-backend/
├── services/
│   ├── auth-service/          # Auth, sesi login, rotasi token RS256, OTP, role RBAC
│   └── api-gateway/           # Reverse proxy, verifikasi JWKS JWT, rate limiting, anti-spoofing
├── pkg/                       # Shared modules (response, requestid, logger, gatewayauth)
├── certs/                     # RSA private & public keys (jwt_private.pem & jwt_public.pem)
├── deploy/                    # docker-compose.yml & prometheus.yml
├── docs/                      # API.md, DECISIONS.md, Postman collection & environment
├── scripts/                   # e2e.sh, genkeys, dan scripts/infra
├── Makefile                   # Otomasi build, test, seed, migrate
└── README.md
```

---

## 3. Cara Menjalankan dari Clone Bersih (Clean Setup)

### Langkah 1: Clone Repositori
```bash
git clone https://github.com/diaricakarsa/pmb-backend.git
cd pmb-backend
```

### Langkah 2: Salin File Konfigurasi Environment
```bash
cp services/auth-service/.env.example services/auth-service/.env
cp services/api-gateway/.env.example services/api-gateway/.env
```

### Langkah 3: Generate Pasangan Kunci RSA (RS256)
Sistem autentikasi membutuhkan kunci privat dan publik RSA untuk penandatanganan token JWT. Gunakan script yang telah disediakan:
```bash
go run ./scripts/genkeys
```
Script ini akan menghasilkan file:
- `certs/jwt_private.pem` (Kunci privat untuk Auth Service)
- `certs/jwt_public.pem` (Kunci publik yang disajikan via endpoint JWKS)

*(Atau alternatif manual via OpenSSL)*:
```bash
mkdir -p certs
openssl genrsa -out certs/jwt_private.pem 2048
openssl rsa -in certs/jwt_private.pem -pubout -out certs/jwt_public.pem
```

### Langkah 4: Jalankan Seluruh Sistem dengan Docker Compose
Pastikan Docker Desktop / daemon aktif:
```bash
docker compose -f deploy/docker-compose.yml up -d --build
```
Verifikasi bahwa seluruh container berjalan sehat (*healthy*):
```bash
docker compose -f deploy/docker-compose.yml ps
```

---

## 4. Akun Default Superadmin (Development Seed)

Saat `auth-service` pertama kali berjalan, sistem secara otomatis mengeksekusi migrasi database dan menyuntikkan (*seed*) akun superadmin awal:
- **Email**: `superadmin@kampus.ac.id`
- **Password**: `Superadmin123!`
- **Role**: `superadmin`

> [!NOTE]
> Akun ini dapat digunakan untuk login via `POST /api/v1/auth/login` guna memperoleh token ber-role `superadmin`, yang berhak mengelola pengguna internal di endpoint `/api/v1/auth/admin/*`.

---

## 5. Port & Layanan yang Tersedia

| Layanan | Port Host | Keterangan |
|---|---|---|
| **API Gateway** | `http://localhost:8000` | Pintu gerbang utama semua request klien (`/api/v1/...`) |
| **Swagger UI** | `http://localhost:8000/api/v1/auth/docs/index.html` (atau `http://localhost:8080/docs/index.html`) | Dokumentasi interaktif OpenAPI |
| **Mailpit Web UI** | `http://localhost:8025` | Dasbor penangkap email OTP & Link Reset Password |
| **Mailpit SMTP** | `localhost:1025` | Port SMTP internal untuk pengiriman email |
| **PostgreSQL** | `localhost:5432` | Database `pmb_auth` (User: `pmb_user`, Pass: `pmb_password`) |
| **Redis** | `localhost:6379` | Sesi, token denylist, dan sliding window rate limiter |

---

## 6. Pengujian End-to-End (`e2e.sh`)

Skrip `scripts/e2e.sh` menguji seluruh alur bisnis calon mahasiswa, otorisasi role admin, serta skenario negatif keamanan (OTP lockout, password lockout, token reuse, anti-spoofing, reset password).

### A. Menjalankan di Linux / macOS
```bash
# Pastikan jq terpasang (apt install jq / brew install jq)
bash scripts/e2e.sh
```

### B. Menjalankan di Windows (Wajib Menggunakan Git Bash)
Di Windows, pastikan skrip dieksekusi menggunakan binary **Git Bash**, BUKAN shell WSL launcher:

1. **Instalasi `jq` (jika belum ada)**:
   ```powershell
   winget install jqlang.jq
   # atau via choco
   choco install jq
   ```
   *(Catatan: `scripts/e2e.sh` juga dilengkapi fallback regex/grep sehingga tetap dapat berjalan jika jq belum terpasang).*

2. **Jalankan via PowerShell**:
   ```powershell
   & "C:\Program Files\Git\bin\bash.exe" scripts/e2e.sh
   ```
   Atau buka terminal **Git Bash** langsung dan jalankan:
   ```bash
   bash scripts/e2e.sh
   ```

---

## 7. Catatan Infrastruktur Tiruan (`scripts/infra`)

> [!WARNING]
> Folder `scripts/infra/` adalah emulator lokal in-memory (`miniredis` dan `embedded-postgres`) yang **hanya ditujukan untuk development darurat ketika Docker tidak tersedia**.
> **Folder ini bukan pengganti pengujian di Docker.** Seluruh tes verifikasi akhir wajib dieksekusi terhadap container Docker Compose resmi.
