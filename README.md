# PMB Backend — Sistem Penerimaan Mahasiswa Baru

Repositori monorepo backend Sistem PMB (Fase 1: **Auth Service** dan **API Gateway**).

## Tech Stack
- **Language**: Go 1.25+
- **Framework**: Gin
- **Database**: PostgreSQL 15, GORM, `golang-migrate`
- **Cache & Session**: Redis 7
- **Email Development**: Mailpit
- **Containerization**: Docker Compose

## Struktur Monorepo
```
pmb-backend/
├── services/
│   ├── auth-service/          # Auth, identitas akun, sesi, token RS256, OTP, role
│   └── api-gateway/           # Reverse proxy, validasi JWT via JWKS, rate limiting, CORS
├── pkg/                       # Shared modules (response, requestid, logger, gatewayauth)
├── deploy/                    # docker-compose.yml & prometheus.yml
├── docs/                      # DECISIONS.md, API.md, dan diagram use case / flowchart
├── Makefile                   # Perintah build, test, run, lint
├── go.work                    # Go workspace lokal
└── README.md
```

## Memulai Cepat (Local Development)

### 1. Prasyarat
- Go 1.25 atau lebih baru
- Docker & Docker Compose

### 2. Konfigurasi Lingkungan
Salin file template `.env.example`:
```bash
cp services/auth-service/.env.example services/auth-service/.env
cp services/api-gateway/.env.example services/api-gateway/.env
```

### 3. Menjalankan Infrastruktur (Postgres, Redis, Mailpit)
```bash
make up
```
Atau menggunakan Docker Compose langsung:
```bash
docker compose -f deploy/docker-compose.yml up -d postgres redis mailpit
```

- **PostgreSQL**: `localhost:5432`
- **Redis**: `localhost:6379`
- **Mailpit Web UI**: `http://localhost:8025` (melihat email OTP/reset password pengujian)

### 4. Build, Test, dan Vet
```bash
make build
make vet
make test
```
