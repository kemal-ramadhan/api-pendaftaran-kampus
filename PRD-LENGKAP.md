# PRD LENGKAP — Sistem Pendaftaran Mahasiswa Baru (PMB)

| | |
|---|---|
| **Produk** | Sistem PMB, modul pertama dari "Sistem Kampus" |
| **Lead / Product Owner** | Pak Kemal Ramadhan (pkm.asia) |
| **Backend** | Diar — Go, Gin, microservice |
| **Frontend** | Wahhed — Vue 3 murni (dikerjakan terpisah, bukan scope dokumen ini) |
| **Repository** | `https://github.com/diaricakarsa/pmb-backend` (private, branch `main`) |
| **Versi dokumen** | Konsolidasi v3 — 24 September 2026 |
| **Isi** | Bagian A: konteks & status · Bagian B: PRD Fase 1 (Auth + Gateway) · Bagian C: PRD Fase 2 (semua service PMB + OCR KTP) · Bagian D: Change Request (Duitku) · Bagian E: keputusan tambahan & masalah yang diketahui |

> **Dokumen ini adalah satu-satunya sumber kebenaran.** Dokumen ini menggantikan `PRD.md`, `PRD-pmb-auth-gateway-v2.md`, dan `PRD-fase-2.md`. Di dalam Bagian B dan C masih ada rujukan lama; baca dengan pemetaan berikut:
> - "PRD.md Bagian X" atau "PRD fase 1 Bagian X" → **Bagian B, nomor X**
> - "PRD-fase-2.md Bagian X" → **Bagian C, nomor X**
> - Jika Bagian D atau E bertentangan dengan Bagian B/C, **Bagian D/E yang berlaku** (lebih baru).

---

# BAGIAN A — KONTEKS & STATUS SAAT INI

## A.1 Tujuan produk

Seluruh alur PMB berjalan end-to-end lewat API Gateway:

> Registrasi akun → verifikasi OTP → pilih gelombang, jalur, prodi → bayar pendaftaran (Duitku) → isi biodata (bisa scan KTP dengan OCR) → upload dokumen → verifikasi admin → jadwal & kartu ujian → seleksi (tes/wawancara/berkas) → rekomendasi prodi → persetujuan pimpinan → pengumuman → bayar daftar ulang (lunas/cicilan) → daftar ulang → NIM & akun mahasiswa.

Aktor: Calon Mahasiswa, Admin PMB, Keuangan, Dosen, Prodi, Pimpinan Universitas, Superadmin, Payment Gateway (Duitku). Use case UC-01 s/d UC-33 (+ UC-06a Scan KTP) — lihat Bagian C.3.

Dokumen desain di repo: `docs/flowchart-sistem-pmb.pdf`, `docs/use-case-sistem-pmb.pdf`, `docs/schema-dan-class-diagram.md`, `docs/class-diagram-auth-service.md`.

## A.2 Status per 24 September 2026

| Area | Status | Bukti / catatan |
|---|---|---|
| Auth Service | ✅ **Terverifikasi** | e2e 43/43 PASS di Docker dari DB kosong (19 Sep) |
| API Gateway | ✅ **Terverifikasi** | termasuk perbaikan bypass gateway (`GATEWAY_SIGNATURE_INVALID`) |
| Organization, Identity, Notification, Admission, Finance, Exam, Scoring, Selection, Student, NIM, OCR | ⚠️ **Kode ada, BELUM terverifikasi** | Agent sebelumnya melaporkan tahap 1–11 selesai, tetapi `docker compose up --build` **gagal** (lihat A.4). Semua service ini wajib diaudit dan diverifikasi ulang |
| Payment gateway | 🔄 **Berubah** | Lead memutuskan **Duitku** (CR-001, Bagian D) |
| Frontend | Dikerjakan terpisah oleh Wahhed | Kontrak API di `docs/API.md` |

## A.3 Lingkungan pengembang

- OS: **Windows**, PowerShell. Script bash dijalankan dengan Git Bash: `& "C:\Program Files\Git\bin\bash.exe" scripts/e2e.sh`.
- Docker Desktop (harus dinyalakan sebelum `docker compose`).
- Compose file: `deploy/docker-compose.yml`.
- Gateway: `http://localhost:8000/api/v1/...`, Mailpit: `http://localhost:8025`.
- Superadmin development: `superadmin@kampus.ac.id` / `Superadmin123!` (dari env `SEED_SUPERADMIN_*`).
- Line ending: file `.sh` wajib LF (`.gitattributes`: `*.sh text eol=lf`).

## A.4 Masalah terbuka yang harus diselesaikan lebih dulu

1. **Build Docker gagal** di `scoring-service`: `missing go.sum entry` untuk gin, gorm, zap, uuid, godotenv, excelize. Kemungkinan besar service lain juga (build lain ter-cancel). Penyebab: `go mod tidy` tidak dijalankan per module.
2. **Versi Go tidak seragam**: sebagian Dockerfile memakai `golang:1.26-alpine`, `student-service` memakai `golang:1.25-alpine`. Seragamkan (versi di `go.mod` dan Dockerfile harus konsisten di semua module).
3. E2E yang tampak "berjalan" setelah build gagal **tidak valid** (menembak container lama / tidak ada container). Hanya e2e setelah semua container **healthy** yang dihitung.
4. Klaim "tahap 1–11 selesai" belum pernah dibuktikan dengan e2e penuh flowchart 1 dari database kosong.

---

# BAGIAN B — PRD FASE 1: AUTH SERVICE & API GATEWAY (sudah terverifikasi)

> Rujukan "Bagian X" di dalam bagian ini = Bagian B nomor X.

## (Judul asli) PRD Fase 1 — Auth Service & API Gateway

| | |
|---|---|
| **Produk** | Sistem Pendaftaran Mahasiswa Baru (PMB), modul pertama dari "Sistem Kampus" |
| **Lingkup dokumen** | Backend fase 1: **Auth Service** dan **API Gateway** |
| **Stack** | Go 1.25, Gin, GORM + PostgreSQL 15, Redis 7, Docker Compose |
| **Arsitektur** | Microservice, semua traffic klien lewat API Gateway |
| **Referensi kode** | `reference/jatidiri-auth-service` (auth service yang sudah jalan di produk lain, dipakai sebagai basis) |
| **Dokumen pendukung** | `reference/EduPilot.postman_collection.json` (contoh API auth lama), `docs/flowchart-sistem-pmb.pdf`, `docs/use-case-sistem-pmb.pdf` |
| **Target waktu** | Auth Service 18–25 Sep 2026, API Gateway 24–30 Sep 2026 |
| **Status** | Draft v2 (final untuk eksekusi) |

### Perubahan dari v1
- Ditambahkan **batas tanggung jawab service** (Bagian 2.3): Auth hanya mengurus akun, kredensial, sesi, token, dan role. Biodata lengkap milik Identity Service, data prodi/fakultas milik Organization Service (keduanya dibangun di fase berikutnya).
- Nama endpoint **diselaraskan dengan API auth lama** (Postman EduPilot/Jatidiri) supaya familiar bagi tim: `/auth/verify`, `PUT /auth/profile`, `PUT /auth/password`, `/auth/health`. Perilakunya tetap mengikuti PRD ini (Bagian 5.9).
- Ditambahkan deliverable **Postman collection** PMB.
- Slot routing gateway disesuaikan dengan daftar service di timeline tim.
- Pertanyaan terbuka v1 sudah diputuskan (Bagian 13).

---

## 0. Instruksi untuk AI Agent (fase 1 — SUDAH DIEKSEKUSI; berlaku sebagai aturan umum)

1. Kerjakan milestone di **Bagian 11** secara berurutan. Jangan lompat ke milestone berikutnya sebelum acceptance criteria milestone saat ini terpenuhi dan `go build ./...`, `go vet ./...`, `go test ./...` lulus.
2. `reference/jatidiri-auth-service` adalah **referensi**, bukan target edit. Salin pola dan kode yang berguna ke `services/auth-service`, lalu refactor sesuai PRD ini. Jangan mengubah folder referensi.
3. **Jangan pernah** menyalin atau commit file `.env` asli, folder `postgres-data/`, atau secret apa pun. Buat `.env.example` berisi placeholder. Tambahkan `.env`, `postgres-data/`, `*.pem` ke `.gitignore`.
4. Jika PRD ini ambigu atau bertentangan dengan kode referensi, **PRD yang menang**. Jika PRD tidak mengatur sesuatu, pilih solusi paling sederhana yang konsisten dengan pola kode referensi, lalu catat keputusannya di `docs/DECISIONS.md`.
5. Bahasa: identifier kode, nama tabel, dan field JSON dalam **bahasa Inggris**. Pesan error untuk user (`message`) dalam **bahasa Indonesia**.
6. Setiap endpoint wajib punya anotasi Swagger (swaggo) dan tercantum di `docs/`.
7. `reference/EduPilot.postman_collection.json` adalah contoh API auth lama. Pakai untuk memahami gaya dan penamaan, tetapi **perilaku dan format respons mengikuti PRD ini** (lihat 5.9).

---

## 1. Latar Belakang

Kampus membutuhkan sistem PMB yang mengelola calon mahasiswa, dokumen, pembayaran (termasuk cicilan), seleksi, sampai daftar ulang, dengan akses untuk banyak role (calon mahasiswa, admin PMB, keuangan, dosen, prodi, pimpinan, superadmin). Flowchart dan use case diagram sudah disetujui (lihat `docs/flowchart-sistem-pmb.pdf` dan `docs/use-case-sistem-pmb.pdf` bila disertakan).

Backend dipecah per domain menjadi microservice (lihat daftar lengkap di 2.3). Fase 1 hanya membangun dua fondasi yang dibutuhkan semua service lain:

- **Auth Service**: identitas, login, registrasi, OTP, reset password, role & hak akses.
- **API Gateway**: satu pintu masuk untuk frontend, meneruskan request ke service yang tepat, dan menjadi lapisan keamanan bersama.

Frontend (Vue 3 murni) dikerjakan terpisah. Kebutuhan pertama frontend: halaman Login, Register, Verifikasi OTP, Lupa Password, Reset Password, dan Dashboard dummy yang menyimpan token, **khusus role Calon Mahasiswa**. API di PRD ini harus langsung bisa melayani halaman-halaman tersebut.

## 2. Tujuan & Non-Tujuan

### 2.1 Tujuan
- Auth service yang mendukung **banyak role** dan bisa dipakai semua service kampus berikutnya.
- Calon mahasiswa bisa daftar sendiri, verifikasi OTP email, login, refresh token, logout, lupa & reset password.
- Akun internal (admin PMB, keuangan, dosen, prodi, pimpinan, superadmin) dibuat dan dikelola oleh superadmin.
- API Gateway yang merutekan `/api/v1/<service>/...` ke service terkait, memvalidasi JWT, memeriksa role per route, rate limiting, CORS, dan meneruskan identitas user ke service hilir secara aman.
- Seluruh stack bisa dijalankan dengan satu perintah `docker compose up`.

### 2.2 Non-Tujuan (fase ini)
- Service pendaftaran, berkas, pembayaran, seleksi, dan notifikasi (hanya disiapkan slot routing-nya di gateway).
- Frontend.
- Login Google (kode referensi ada, tetapi **dimatikan** lewat feature flag, lihat 5.7).
- OTP via WhatsApp/SMS (fase ini email saja, tetapi desain OTP harus bisa ditambah channel lain).
- Two-factor authentication.
- Foto profil, biodata lengkap (NIK, tempat/tanggal lahir, alamat, orang tua, dll.) → **Identity Service**.
- Master data fakultas, prodi, dan unit kerja → **Organization Service**.

### 2.3 Batas Tanggung Jawab Service

Daftar service jangka panjang (dari timeline tim): Gateway, Auth, Identity, Organization, Admission, Finance, Selection, Exam, Scoring, Student, NIM. Fase ini hanya **Gateway** dan **Auth**, tetapi batasnya harus jelas sejak awal agar Auth tidak membengkak.

| Service | Pemilik data | Fase |
|---|---|---|
| **Auth** | Akun login (email, phone, password hash, status, verifikasi), sesi & token, OTP, reset password, **role & scope**, audit log autentikasi. Menyimpan `full_name` hanya sebagai *display name* untuk email dan token | **Sekarang** |
| **API Gateway** | Tidak punya database. Routing, verifikasi token, rate limit, CORS | **Sekarang** |
| Identity | Biodata lengkap orang (calon mahasiswa, mahasiswa, dosen, staf), foto profil | Berikutnya |
| Organization | Fakultas, prodi, unit, kuota prodi | Berikutnya |
| Admission, Finance, Selection, Exam, Scoring, Student, NIM | Sesuai use case PMB | Berikutnya |

Aturan integrasi:
- **`users.id` (UUID) dari Auth adalah ID global orang di seluruh sistem.** Service lain menyimpannya sebagai `user_id` tanpa foreign key lintas database.
- `scope_id` pada role (mis. kode prodi `TI`) disimpan sebagai string di Auth tanpa foreign key. Validasi terhadap Organization Service ditambahkan nanti; fase ini cukup validasi format (huruf besar/angka/underscore, 2–20 karakter).
- Tidak ada service yang membaca database service lain. Komunikasi lewat HTTP internal (`/internal/*`) atau, nanti, event.
- Saat Identity Service dibuat, `full_name` di Auth tetap ada sebagai cache display name dan diperbarui lewat internal API.

## 3. Struktur Repository

Monorepo:

```
pmb-backend/
├── services/
│   ├── auth-service/
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── config/        # env, DB, Redis, logger, tracer
│   │   │   ├── model/         # entitas GORM
│   │   │   ├── repository/    # akses DB (baru; referensi query langsung di service)
│   │   │   ├── service/       # logika bisnis
│   │   │   ├── controller/    # handler HTTP (Gin)
│   │   │   ├── middleware/
│   │   │   ├── route/
│   │   │   ├── dto/           # request/response struct
│   │   │   └── utils/
│   │   ├── migrations/        # SQL migration (golang-migrate) + seeder role & superadmin
│   │   ├── docs/              # swagger
│   │   ├── Dockerfile
│   │   └── .env.example
│   └── api-gateway/
│       ├── cmd/main.go
│       ├── internal/{config,proxy,middleware,jwks,route,health}
│       ├── config/routes.yaml
│       ├── Dockerfile
│       └── .env.example
├── pkg/                        # kode bersama opsional (response format, request id, logger)
├── deploy/
│   ├── docker-compose.yml
│   └── prometheus.yml
├── docs/
│   ├── DECISIONS.md
│   └── API.md                  # ringkasan kontrak API untuk tim frontend
├── Makefile                    # run, test, lint, migrate, swag
└── README.md
```

Module Go: `github.com/<org>/pmb-backend/services/auth-service` dan `.../api-gateway` (org dijadikan variabel di README, default `pkm-asia`). Jangan memakai module path referensi (`iamhexagoninc/go-jatidiri-app`).

## 4. Role & Hak Akses

### 4.1 Daftar Role (seed awal)

| Kode | Nama | Cara akun dibuat |
|---|---|---|
| `calon_mahasiswa` | Calon Mahasiswa | Registrasi mandiri |
| `mahasiswa` | Mahasiswa | Ditambahkan oleh sistem (endpoint internal) setelah NIM terbit. Pada fase ini cukup disediakan endpoint-nya |
| `admin_pmb` | Admin PMB | Superadmin |
| `keuangan` | Keuangan | Superadmin |
| `dosen` | Dosen | Superadmin |
| `prodi` | Prodi (Kaprodi/admin prodi) | Superadmin, wajib `scope` prodi |
| `pimpinan` | Pimpinan Universitas | Superadmin |
| `superadmin` | Superadmin | Seeder (dari env) dan superadmin lain |

### 4.2 Aturan
- Satu user boleh punya **lebih dari satu role** (contoh: dosen yang juga `prodi`).
- Role `prodi` (dan ke depan role lain yang terikat unit) punya **scope**: `scope_type` (mis. `prodi`) dan `scope_id` (kode prodi, string, milik Organization Service, lihat 2.3). Scope ikut masuk ke JWT.
- Role tidak boleh diubah oleh user itu sendiri. Hanya superadmin (via admin API) atau service internal (via internal API).
- Role disimpan di tabel, bukan hardcode di kode, supaya bisa ditambah tanpa deploy ulang. Kode tetap boleh punya konstanta untuk role inti.
- Registrasi publik **selalu** menghasilkan role `calon_mahasiswa`. Field `role` dari klien diabaikan.

## 5. Auth Service — Kebutuhan Fungsional

### 5.1 Model Data (PostgreSQL)

Gunakan UUID (v7 jika tersedia, jika tidak v4) sebagai primary key. Kelola skema dengan **migration SQL** (`golang-migrate`), bukan `AutoMigrate`.

**users**
| Kolom | Tipe | Catatan |
|---|---|---|
| id | uuid PK | |
| email | varchar(255) unique, not null | disimpan lowercase + trim |
| phone | varchar(20) null | format `08…`/`+62…`, dinormalisasi ke `+62…` |
| password_hash | varchar(255) not null | bcrypt, cost dari env (default 12) |
| full_name | varchar(255) not null | |
| status | varchar(20) not null default `pending` | `pending` (belum verifikasi), `active`, `inactive` (dinonaktifkan admin) |
| email_verified_at | timestamptz null | |
| last_login_at | timestamptz null | |
| password_changed_at | timestamptz null | |
| created_at, updated_at | timestamptz | |
| deleted_at | timestamptz null | soft delete |

**roles**: `id` (smallserial), `code` (unique), `name`, `description`, `created_at`.

**user_roles**: `user_id` FK, `role_id` FK, `scope_type` varchar(30) null, `scope_id` varchar(50) null, `created_at`, `created_by` uuid null. Unique `(user_id, role_id, scope_type, scope_id)`.

**user_otps**: `id`, `user_id` FK, `purpose` (`email_verification`; siapkan enum untuk `login`, `phone_verification`), `channel` (`email`), `code_hash` (SHA-256 dari kode + secret, **bukan** plaintext), `expires_at`, `attempts` smallint default 0, `max_attempts` smallint default 5, `consumed_at` null, `created_at`.

**password_resets**: `id`, `user_id` FK, `token_hash` (SHA-256), `expires_at`, `used_at` null, `created_at`.

**audit_logs**: `id` bigserial, `user_id` null, `actor_id` null, `action` (mis. `auth.login.success`, `auth.login.failed`, `user.role.assigned`), `ip`, `user_agent`, `metadata` jsonb, `created_at`.

Seeder: semua role di 4.1 dan satu akun superadmin dari env `SEED_SUPERADMIN_EMAIL`, `SEED_SUPERADMIN_PASSWORD` (idempotent, dilewati bila sudah ada).

### 5.2 Token

- **Algoritma: RS256.** Auth service memegang private key; public key dipublikasikan di `GET /.well-known/jwks.json` (dengan `kid`). Gateway dan service lain memverifikasi token tanpa perlu secret bersama. Key dibaca dari file path di env; jika tidak ada dan `APP_ENV=development`, generate key sementara saat start dan log peringatan.
- **Access token**: TTL default 15 menit (env). Claims:
  ```json
  {
    "iss": "pmb-auth-service",
    "sub": "<user uuid>",
    "aud": "pmb-api",
    "exp": 0, "iat": 0, "jti": "<uuid>",
    "typ": "access",
    "email": "user@mail.com",
    "name": "Nama Lengkap",
    "roles": ["calon_mahasiswa"],
    "scopes": [{"role": "prodi", "type": "prodi", "id": "TI"}]
  }
  ```
- **Refresh token**: TTL default 7 hari, `typ: "refresh"`, berisi `sub`, `jti`, `sid` (session id). Disimpan di Redis `auth:refresh:{sub}:{sid}` → `jti`. Mendukung **multi-sesi** (beberapa device).
- **Rotasi refresh token**: setiap `/refresh` menerbitkan refresh token baru dan meng-overwrite `jti` di Redis. Jika refresh token lama (jti tidak cocok) dipakai lagi, anggap token dicuri: hapus sesi tersebut dan balas 401.
- Token dengan `typ` salah wajib ditolak (refresh token tidak boleh dipakai sebagai access token dan sebaliknya).
- **Logout**: hapus sesi di Redis dan masukkan `jti` access token ke denylist Redis `auth:denylist:{jti}` dengan TTL = sisa umur token. Gateway memeriksa denylist ini.
- Logout semua sesi (`/logout-all`) menghapus semua `auth:refresh:{sub}:*`.
- Saat password diganti/di-reset atau user dinonaktifkan: cabut semua sesi.

### 5.3 Endpoint

Path di bawah adalah path **di dalam auth service**. Lewat gateway, prefix-nya menjadi `/api/v1/auth` (lihat 6.2).

#### Publik
| Method | Path | Deskripsi |
|---|---|---|
| POST | `/auth/register` | Registrasi calon mahasiswa |
| POST | `/auth/verify` | Verifikasi OTP email (tanpa token), sekaligus auto-login |
| POST | `/auth/resend-otp` | Kirim ulang OTP |
| POST | `/auth/login` | Login |
| POST | `/auth/refresh` | Tukar refresh token dengan pasangan token baru |
| POST | `/auth/forgot-password` | Kirim link reset password |
| POST | `/auth/reset-password` | Set password baru dengan token reset |
| GET | `/.well-known/jwks.json` | Public key JWT |
| GET | `/health`, `/health/ready`, `/auth/health` | Liveness & readiness (cek DB + Redis). `/auth/health` alias agar bisa dicek lewat gateway |

#### Terautentikasi (semua role)
| Method | Path | Deskripsi |
|---|---|---|
| GET | `/auth/me` | Profil user + roles |
| PUT | `/auth/profile` | Ubah data akun milik sendiri saja: `full_name`, `phone`. Tidak ada varian `/:id` |
| PUT | `/auth/password` | Ganti password (wajib `current_password`), cabut sesi lain |
| POST | `/auth/logout` | Logout sesi saat ini |
| POST | `/auth/logout-all` | Logout semua sesi |

#### Admin (role `superadmin`)
| Method | Path | Deskripsi |
|---|---|---|
| GET | `/auth/admin/roles` | Daftar role |
| GET | `/auth/admin/users` | List user, filter `role`, `status`, `q` (nama/email), paginasi `page`, `per_page` (max 100) |
| POST | `/auth/admin/users` | Buat user internal (email, full_name, phone, roles[], password awal **atau** kirim link set-password via email). Status langsung `active`, email dianggap terverifikasi |
| GET | `/auth/admin/users/:id` | Detail user |
| PUT | `/auth/admin/users/:id` | Ubah data dasar |
| PUT | `/auth/admin/users/:id/roles` | Ganti set role (+scope) user |
| PATCH | `/auth/admin/users/:id/status` | `active` / `inactive` (inactive → cabut semua sesi) |
| POST | `/auth/admin/users/import` | Import user internal via CSV (email, full_name, phone, role, scope_id). Balas ringkasan sukses/gagal per baris |

Superadmin tidak boleh menonaktifkan atau mencabut role superadmin dirinya sendiri, dan sistem harus selalu menyisakan minimal satu superadmin aktif.

#### Internal (antar service, **tidak diekspos** gateway)
Dilindungi header `X-Internal-Token` (secret dari env) **dan** hanya di-mount bila request datang dari jaringan internal Docker. Gateway wajib memblokir `/internal/*` dari luar.
| Method | Path | Deskripsi |
|---|---|---|
| GET | `/internal/users/:id` | Data ringkas user |
| POST | `/internal/users/batch` | Body `{ "ids": [...] }`, max 500 |
| POST | `/internal/users/:id/roles` | Tambah role (dipakai service pendaftaran untuk memberi role `mahasiswa`) |

### 5.4 Detail Alur

**Register** — body: `full_name`, `email`, `phone`, `password`, `password_confirmation`.
1. Validasi: email valid; password min 8 karakter, mengandung huruf dan angka; konfirmasi cocok; phone format Indonesia.
2. Jika email sudah terdaftar **dan terverifikasi** → 409 `EMAIL_TAKEN`. Jika terdaftar tapi masih `pending` → timpa data (nama, phone, password) dan kirim OTP baru. Ini mencegah akun "terkunci" karena pendaftar salah ketik lalu menutup halaman.
3. Buat user `status=pending`, role `calon_mahasiswa`, buat OTP 6 digit (pakai `crypto/rand`, **bukan** `math/rand`), berlaku 10 menit, kirim email.
4. Respons 201: `{ "email": "...", "otp_expires_at": "...", "resend_available_at": "..." }`. **Tidak** menerbitkan token.
5. Field `role` dan `is_import` dari klien diabaikan.

**Verify OTP** — body: `email`, `otp`.
1. Ambil OTP aktif terbaru (`purpose=email_verification`, belum consumed).
2. Jika kedaluwarsa → 400 `OTP_EXPIRED`. Jika `attempts >= max_attempts` → 429 `OTP_LOCKED` (harus resend).
3. Jika salah → **increment `attempts`**, balas 400 `OTP_INVALID` dengan `remaining_attempts`.
4. Jika benar → tandai consumed, set `email_verified_at`, `status=active`, terbitkan access + refresh token (auto-login). Respons sama dengan login.

**Resend OTP** — body: `email`. Cooldown 60 detik per email (Redis), maksimal 5 kali per jam. OTP lama dinonaktifkan. Jika email tidak ada atau sudah terverifikasi, tetap balas 200 dengan pesan generik (cegah enumerasi).

**Login** — body: `email`, `password`. Siapkan struktur agar ke depan bisa login pakai NIM/NIDN (field `identifier`), tapi fase ini email saja.
- Email tidak ada **atau** password salah → 401 `INVALID_CREDENTIALS` dengan pesan yang **sama** (cegah enumerasi).
- Status `pending` → 403 `EMAIL_NOT_VERIFIED` beserta `email`, supaya frontend bisa mengarahkan ke halaman OTP (dan auth service mengirim OTP baru bila OTP lama kedaluwarsa).
- Status `inactive` → 403 `ACCOUNT_INACTIVE`.
- Lockout: 5 kali gagal dalam 15 menit per email → kunci 15 menit (Redis), balas 429 `TOO_MANY_ATTEMPTS`.
- Sukses → update `last_login_at`, catat audit log, balas token.

Respons sukses login / verify / refresh:
```json
{
  "success": true,
  "message": "Login berhasil",
  "data": {
    "access_token": "...",
    "refresh_token": "...",
    "token_type": "Bearer",
    "expires_in": 900,
    "user": {
      "id": "uuid", "email": "...", "full_name": "...", "phone": "+62...",
      "roles": ["calon_mahasiswa"], "scopes": [], "status": "active"
    }
  }
}
```

**Forgot Password** — body: `email`. Selalu 200 dengan pesan generik. Jika user ada dan aktif: buat token acak 32 byte (`crypto/rand`, base64url), simpan **hash**-nya, berlaku 30 menit, kirim link `${FRONTEND_URL}/reset-password?token=...`. Cooldown 60 detik per email.

**Reset Password** — body: `token`, `password`, `password_confirmation`. Token tidak valid/kedaluwarsa/sudah dipakai → 400 `RESET_TOKEN_INVALID`. Sukses → update password, tandai token used, cabut semua sesi.

### 5.5 Email
- Pakai SMTP (gomail, seperti referensi). Template HTML sederhana untuk OTP dan reset password, dengan nama kampus dari env `APP_NAME`.
- Pengiriman email tidak boleh memblokir respons terlalu lama: kirim di goroutine dengan timeout dan log error. Buat interface `Mailer` supaya bisa diganti (mis. mock untuk test, atau layanan notifikasi terpisah nanti).
- `MAIL_DRIVER=log` untuk development: email tidak dikirim, isinya (termasuk OTP) ditulis ke log.

### 5.6 Format Respons & Error (berlaku untuk semua service)

Sukses:
```json
{ "success": true, "message": "…", "data": { }, "meta": { "page": 1, "per_page": 20, "total": 120 } }
```
Error:
```json
{ "success": false, "message": "Pesan untuk user (Bahasa Indonesia)", "error_code": "OTP_INVALID",
  "errors": { "email": ["Format email tidak valid"] }, "request_id": "…" }
```
- `errors` hanya untuk error validasi (422), berisi pesan per field.
- `error_code` stabil dan terdokumentasi di `docs/API.md` (frontend memakainya untuk logika tampilan).
- Jangan pernah membocorkan error internal (SQL, stack trace) ke klien; log saja dengan `request_id`.

Letakkan helper respons di `pkg/response` agar dipakai ulang service lain.

### 5.7 Yang Diambil dari Kode Referensi & Yang Harus Diperbaiki

**Pertahankan polanya:** struktur `cmd/` + `internal/`, Gin, GORM, Redis, zap logger, OpenTelemetry + Jaeger, Prometheus (`ginprom`), swaggo, rate limiter berbasis Redis, request ID middleware, secure headers, graceful shutdown, Dockerfile multi-stage dengan user non-root.

**Buang / ganti:**
- Domain `customer` dan endpoint `/customer/*`, `/other/*` (spesifik produk lain). Data profil pindah ke `users`.
- Role hardcode `students/teachers/...` → tabel `roles` + `user_roles`.
- Login Google: pindahkan ke belakang feature flag `GOOGLE_LOGIN_ENABLED=false` (boleh tidak diimplementasikan di fase ini).
- `AutoMigrate` → migration SQL.

**Celah keamanan di referensi yang WAJIB tidak terulang:**
1. `POST /auth/register` menerima `is_import=true` dari publik sehingga verifikasi email terlewati.
2. Percobaan OTP tidak pernah di-increment, jadi batas percobaan tidak berfungsi.
3. OTP dan token reset memakai/disimpan tanpa hash; OTP memakai `math/rand`.
4. `PUT /auth/change-role` membolehkan user mengubah role-nya sendiri.
5. `PUT /auth/profile/:id` membolehkan user mana pun mengubah profil user lain (IDOR).
6. Endpoint `/customer/*` dan `/other/*` tanpa autentikasi.
7. Access token dan refresh token tidak dibedakan (`typ`), memakai secret yang sama, dan tidak membawa role.
8. Pesan login dan lupa password membedakan "user not found" vs "invalid password" (enumerasi akun).
9. Refresh token hanya satu per user (login di device kedua mengeluarkan device pertama) dan tanpa rotasi/deteksi reuse.
10. CORS default `*` bersamaan dengan `Allow-Credentials: true`.
11. `.env` ikut di-`COPY` ke image Docker. Konfigurasi harus lewat environment runtime.

Tulis test yang membuktikan poin 1, 2, 4, 5, 7, dan 8 sudah tertutup.

### 5.8 Kebutuhan Non-Fungsional Auth Service
- Rate limit per route (Redis): register 5/jam/IP, login seperti 5.4, resend-otp dan forgot-password seperti 5.4, refresh 30/5 menit/IP.
- Semua query lewat repository dengan context dan timeout.
- Body JSON maksimal 1 MB; CSV import maksimal 2 MB.
- Password tidak pernah di-log. Log email boleh, tapi masking OTP/token.
- p95 latensi login < 300 ms (di luar waktu bcrypt), endpoint lain < 100 ms pada beban normal.

### 5.9 Pemetaan dari API Auth Lama (Postman EduPilot)

| API lama | API baru (PRD ini) | Perubahan perilaku |
|---|---|---|
| `POST /auth/register` (field `name`, `role`) | `POST /auth/register` | Field `full_name`, `phone`, `password_confirmation`. `role` diabaikan (selalu `calon_mahasiswa`). Tidak lagi mengembalikan token |
| `POST /auth/verify` (butuh Bearer token) | `POST /auth/verify` | Publik, body `email` + `otp`. Sukses → token (auto-login) |
| `POST /auth/resend-otp` (butuh Bearer token) | `POST /auth/resend-otp` | Publik, body `email`, ada cooldown |
| `POST /auth/login` | `POST /auth/login` | Pesan error generik, lockout, token memuat roles |
| `POST /auth/refresh` | `POST /auth/refresh` | Rotasi + deteksi reuse, multi-sesi |
| `POST /auth/logout` | `POST /auth/logout` (+ `/auth/logout-all`) | Access token masuk denylist |
| `GET /auth/me` | `GET /auth/me` | Data akun + roles + scopes (tanpa objek `customer`) |
| `PUT /auth/profile` (birth_place, birth_date, gender) | `PUT /auth/profile` | Hanya `full_name` & `phone`. Biodata lain pindah ke Identity Service |
| `PUT /auth/profile/:id` | Dihapus | Admin memakai `PUT /auth/admin/users/:id` |
| `PUT /auth/photo-profile` | Dihapus | Pindah ke Identity Service |
| `PUT /auth/change-role` | Dihapus | Diganti `PUT /auth/admin/users/:id/roles` (superadmin) |
| `POST /auth/forgot-password`, `/auth/reset-password` | Sama | Token di-hash, pesan generik, cabut semua sesi setelah reset |
| `GET /auth/google/*` | Dimatikan (feature flag) | — |
| `POST /auth/customers/import` | `POST /auth/admin/users/import` | Superadmin only, CSV |
| `/customer/*`, `/other/*` | Dihapus | Kebutuhan antar service lewat `/internal/*`; list user untuk admin lewat `/auth/admin/users` |

Tabel ini juga ditulis ulang di `docs/API.md` bagian "Migrasi dari API lama".

## 6. API Gateway — Kebutuhan Fungsional

### 6.1 Tanggung Jawab
1. **Reverse proxy** berbasis prefix path ke service hilir (`net/http/httputil.ReverseProxy`, tanpa library gateway pihak ketiga).
2. **Autentikasi**: memverifikasi JWT (RS256) dengan JWKS dari auth service (cache, refresh berkala tiap 10 menit dan saat `kid` tidak dikenal). Cek `exp`, `iss`, `aud`, `typ=access`, dan denylist `jti` di Redis.
3. **Otorisasi kasar per route**: route bisa mensyaratkan role tertentu. Otorisasi detail (mis. prodi hanya melihat pendaftar prodinya) tetap tanggung jawab service hilir.
4. **Propagasi identitas**: setelah token valid, gateway menghapus header identitas apa pun yang dikirim klien, lalu menambahkan:
   - `X-User-ID`, `X-User-Email`, `X-User-Roles` (dipisah koma), `X-User-Scopes` (JSON base64url)
   - `X-Request-ID` (pakai dari klien jika ada dan valid, jika tidak generate)
   - `X-Gateway-Signature`: HMAC-SHA256 dari `X-Request-ID + X-User-ID + timestamp` dengan `GATEWAY_SHARED_SECRET`, plus `X-Gateway-Timestamp`. Service hilir menolak request tanpa signature valid (toleransi 60 detik). Sediakan middleware verifikasinya di `pkg/gatewayauth` untuk dipakai semua service (termasuk auth service untuk route non-publik).
   - Header `Authorization` tetap diteruskan.
5. **CORS terpusat**: whitelist origin dari env `CORS_ALLOWED_ORIGINS` (dipisah koma). Service hilir tidak mengatur CORS.
6. **Rate limit global** per IP (Redis, default 120 req/menit) dan override per route.
7. **Batas ukuran body** per route (default 1 MB; siapkan override untuk service berkas nanti, mis. 10 MB).
8. **Timeout** upstream per route (default 10 detik) → 504 jika lewat; upstream mati → 502. Keduanya dalam format error standar.
9. **Blokir** path `/internal/*`, `/metrics`, dan `/docs` service hilir dari luar.
10. **Observability**: log akses terstruktur (method, path, status, latency, upstream, user_id, request_id), metrics Prometheus, propagasi trace OpenTelemetry.
11. **Health**: `GET /health` (gateway hidup) dan `GET /health/ready` (cek setiap upstream `/health`, balas status per service).

### 6.2 Konfigurasi Route (`config/routes.yaml`)

Route didefinisikan di file, bukan di kode, agar service baru cukup menambah entri.

```yaml
services:
  auth:
    url: ${AUTH_SERVICE_URL}          # http://auth-service:8080
    timeout: 10s
  # slot service berikutnya (biarkan ter-comment sampai service-nya dibuat)
  # identity:     { url: ${IDENTITY_SERVICE_URL}, max_body: 5MB }      # /api/v1/identity
  # organization: { url: ${ORGANIZATION_SERVICE_URL} }                 # /api/v1/organization
  # admission:    { url: ${ADMISSION_SERVICE_URL}, max_body: 10MB }    # /api/v1/admission (pendaftaran + dokumen)
  # finance:      { url: ${FINANCE_SERVICE_URL} }                      # /api/v1/finance
  # selection:    { url: ${SELECTION_SERVICE_URL} }                    # /api/v1/selection
  # exam:         { url: ${EXAM_SERVICE_URL} }                         # /api/v1/exam
  # scoring:      { url: ${SCORING_SERVICE_URL} }                      # /api/v1/scoring
  # student:      { url: ${STUDENT_SERVICE_URL} }                      # /api/v1/student
  # nim:          { url: ${NIM_SERVICE_URL} }                          # /api/v1/nim

routes:
  - prefix: /api/v1/auth/.well-known
    service: auth
    rewrite: /.well-known
    auth: public
  - prefix: /api/v1/auth/admin
    service: auth
    rewrite: /auth/admin
    auth: required
    roles: [superadmin]
  - prefix: /api/v1/auth
    service: auth
    rewrite: /auth
    auth: required
    public_paths:                     # sub-path yang tidak butuh token
      - POST /register
      - POST /verify
      - GET /health
      - POST /resend-otp
      - POST /login
      - POST /refresh
      - POST /forgot-password
      - POST /reset-password
    rate_limit: { requests: 60, window: 1m }
```

Konvensi: setiap service mendapat prefix `/api/v1/<nama-service>`. Aturan pencocokan: prefix terpanjang menang. Route yang tidak cocok → 404 format standar. Konfigurasi divalidasi saat start (fail fast jika service tidak dikenal, dsb.).

### 6.3 Urutan Middleware Gateway
Recovery → Request ID → Access log → Tracing → Metrics → Security headers → CORS (termasuk preflight `OPTIONS`) → Blokir path terlarang → Route match → Rate limit → Body limit → Auth (jika bukan publik) → Role check → Strip & inject header → Proxy.

## 7. Kontrak untuk Frontend (Calon Mahasiswa)

Ringkasan ini juga ditulis ulang di `docs/API.md` dengan contoh request/response lengkap.

| Halaman FE | Endpoint (via gateway) | Catatan |
|---|---|---|
| Register | `POST /api/v1/auth/register` | Sukses → arahkan ke halaman OTP membawa `email` |
| Verifikasi OTP | `POST /api/v1/auth/verify`, `POST /api/v1/auth/resend-otp` | Sukses verify → simpan token, masuk dashboard. Tombol kirim ulang aktif setelah `resend_available_at` |
| Login | `POST /api/v1/auth/login` | `EMAIL_NOT_VERIFIED` → arahkan ke halaman OTP |
| Lupa Password | `POST /api/v1/auth/forgot-password` | Selalu tampilkan pesan generik |
| Reset Password | `POST /api/v1/auth/reset-password` | Token dari query string `?token=` |
| Dashboard (dummy) | `GET /api/v1/auth/me` | Contoh pemanggilan terproteksi |
| Semua halaman | `POST /api/v1/auth/refresh` | Dipanggil saat API balas 401 `TOKEN_EXPIRED` |
| Logout | `POST /api/v1/auth/logout` | Body `{ "refresh_token": "..." }` |

Error code minimal yang wajib ada: `VALIDATION_ERROR`, `EMAIL_TAKEN`, `INVALID_CREDENTIALS`, `EMAIL_NOT_VERIFIED`, `ACCOUNT_INACTIVE`, `TOO_MANY_ATTEMPTS`, `OTP_INVALID`, `OTP_EXPIRED`, `OTP_LOCKED`, `OTP_RESEND_COOLDOWN`, `RESET_TOKEN_INVALID`, `TOKEN_MISSING`, `TOKEN_INVALID`, `TOKEN_EXPIRED`, `TOKEN_REVOKED`, `FORBIDDEN`, `NOT_FOUND`, `RATE_LIMITED`, `UPSTREAM_UNAVAILABLE`, `UPSTREAM_TIMEOUT`, `INTERNAL_ERROR`.

Token dikirim di body JSON (frontend menyimpannya sendiri). Desain jangan menutup kemungkinan memindahkan refresh token ke cookie `HttpOnly` nanti.

## 8. Environment Variables

**auth-service** (`.env.example`):
```
APP_NAME="PMB Universitas"
APP_ENV=development
APP_PORT=8080
FRONTEND_URL=http://localhost:5173
DB_HOST=db_auth
DB_PORT=5432
DB_USER=
DB_PASSWORD=
DB_NAME=pmb_auth
REDIS_ADDR=redis:6379
REDIS_PASSWORD=
JWT_PRIVATE_KEY_PATH=/run/secrets/jwt_private.pem
JWT_KID=pmb-auth-1
JWT_ISSUER=pmb-auth-service
JWT_AUDIENCE=pmb-api
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h
BCRYPT_COST=12
OTP_TTL=10m
OTP_SECRET=
INTERNAL_TOKEN=
GATEWAY_SHARED_SECRET=
MAIL_DRIVER=log            # log | smtp
MAIL_HOST=
MAIL_PORT=587
MAIL_USERNAME=
MAIL_PASSWORD=
MAIL_FROM_ADDRESS=
MAIL_FROM_NAME=
SEED_SUPERADMIN_EMAIL=
SEED_SUPERADMIN_PASSWORD=
GOOGLE_LOGIN_ENABLED=false
RATE_LIMIT_ENABLED=true
OTEL_EXPORTER_ENDPOINT=
```

**api-gateway**:
```
APP_ENV=development
APP_PORT=8000
ROUTES_CONFIG=/app/config/routes.yaml
AUTH_SERVICE_URL=http://auth-service:8080
JWKS_URL=http://auth-service:8080/.well-known/jwks.json
JWT_ISSUER=pmb-auth-service
JWT_AUDIENCE=pmb-api
REDIS_ADDR=redis:6379
GATEWAY_SHARED_SECRET=
CORS_ALLOWED_ORIGINS=http://localhost:5173
RATE_LIMIT_GLOBAL=120/1m
TRUSTED_PROXIES=           # kosong = jangan percaya X-Forwarded-For
OTEL_EXPORTER_ENDPOINT=
```

Konfigurasi divalidasi saat start; env wajib yang kosong → service gagal start dengan pesan jelas (kecuali ada default aman untuk development).

## 9. Deployment Lokal

`deploy/docker-compose.yml` berisi: `postgres` (db auth), `redis`, `auth-service`, `api-gateway`, `mailpit` (penangkap email untuk development), dan opsional `jaeger` + `prometheus` via profile `observability`.
- Hanya `api-gateway` (port 8000) dan `mailpit` UI yang di-expose ke host. Auth service tidak di-expose.
- Healthcheck untuk semua container; gateway `depends_on` auth-service `service_healthy`.
- Volume Postgres memakai named volume, bukan bind mount ke path server.
- `make up`, `make down`, `make migrate`, `make seed`, `make test`, `make swag`, `make lint` (golangci-lint).
- Di server, Traefik (yang sudah dipakai sebelumnya) cukup meneruskan domain ke `api-gateway`; routing antar service dilakukan gateway, bukan label Traefik per service.

## 10. Testing

- **Unit test** service layer auth: register (termasuk re-register pending), verify OTP (benar, salah + increment, kedaluwarsa, terkunci), login (sukses, salah, pending, inactive, lockout), refresh (rotasi, reuse detection, typ salah), logout (denylist), reset password, admin role management, guard superadmin terakhir.
- **Unit test** gateway: pencocokan route (prefix terpanjang, public_paths per method), validasi JWT (valid, expired, typ salah, kid tidak dikenal, jti di denylist), role check, strip header spoofing `X-User-*`, blokir `/internal`, CORS preflight, 502/504.
- **Integration test** (build tag `integration`, pakai docker compose atau testcontainers): alur penuh register → OTP (baca dari mailer log/mock) → login → `/me` lewat gateway → refresh → logout → akses `/me` ditolak.
- Test regresi untuk celah di 5.7.
- Target coverage package `service` dan `middleware` ≥ 70%.

## 11. Milestone Eksekusi

**M0 — Setup**
Struktur monorepo, `go.mod` per service, Makefile, `.gitignore`, `.env.example`, docker-compose dengan Postgres + Redis + Mailpit, `pkg/response`, `pkg/requestid`, logger.
*Selesai jika:* `make up` menyalakan infra, `go build ./...` lulus.

**M1 — Auth inti**
Migration + seeder, model, repository, JWT RS256 + JWKS, register, verify, resend-otp, login, refresh (rotasi), logout, logout-all, me, update profile, ganti password, forgot/reset password, mailer (log & smtp), rate limit, audit log, swagger.
*Selesai jika:* semua endpoint publik & terautentikasi di 5.3 berjalan langsung ke auth service dan unit test M1 lulus.

**M2 — Admin & internal**
Admin API (user & role), import CSV, internal API + `X-Internal-Token`, middleware verifikasi signature gateway.
*Selesai jika:* superadmin bisa membuat akun `keuangan` dan `prodi` (dengan scope), dan akun tersebut bisa login dengan role yang benar di token.

**M3 — API Gateway**
Loader & validasi `routes.yaml`, reverse proxy, JWKS cache, auth + denylist, role check, injeksi header + HMAC, CORS, rate limit, body limit, timeout, blokir path, health/ready, metrics, access log.
*Selesai jika:* seluruh alur di Bagian 7 berjalan lewat `http://localhost:8000/api/v1/auth/...` dan unit test gateway lulus.

**M4 — Hardening & dokumentasi**
Integration test end-to-end, test regresi keamanan, `docs/API.md` untuk frontend (termasuk tabel migrasi 5.9), README (cara run, generate key RSA, seed, alur token), `docs/DECISIONS.md`, profile observability.
**Postman collection** `docs/PMB.postman_collection.json`: ikuti gaya `reference/EduPilot.postman_collection.json` (folder per fitur, variabel `{{base_url}}` dan `{{token_access}}`), semua request diarahkan ke gateway (`{{base_url}}=http://localhost:8000/api/v1`), setiap request punya contoh response sukses dan error, serta script test yang otomatis menyimpan `access_token` dan `refresh_token` ke variabel collection setelah login/verify/refresh. Jangan memasukkan email atau password asli; pakai data dummy.
*Selesai jika:* `make test` dan integration test lulus dari clone bersih mengikuti README, dan seluruh request di Postman collection berjalan berurutan (Collection Runner) tanpa gagal.

**Target waktu:** M0–M2 selesai 25 Sep 2026, M3–M4 selesai 30 Sep 2026. Jika tertinggal, prioritaskan alur calon mahasiswa (Bagian 7) dan gateway; import CSV dan profile observability boleh digeser.

## 12. Definition of Done (keseluruhan)
- [ ] Frontend bisa menyelesaikan alur Register → OTP → Dashboard → Logout, serta Lupa & Reset Password, hanya melalui gateway.
- [ ] Token memuat role; superadmin bisa membuat user dengan role apa pun di 4.1.
- [ ] Endpoint admin ditolak (403) untuk non-superadmin, baik di gateway maupun di auth service.
- [ ] Header `X-User-*` palsu dari klien tidak pernah sampai ke service hilir.
- [ ] Ke-11 celah di 5.7 tertutup dan ada test untuk poin yang diminta.
- [ ] Tidak ada secret di repository maupun image Docker.
- [ ] Swagger auth service, `docs/API.md`, dan Postman collection sesuai dengan implementasi.
- [ ] Auth Service tidak menyimpan data yang menurut 2.3 milik Identity/Organization Service.

## 13. Keputusan yang Sudah Diambil

Belum ada konfirmasi dari lead, sehingga keputusan berikut diambil untuk memulai. Semua dicatat juga di `docs/DECISIONS.md` agar mudah diubah.

| Topik | Keputusan | Alasan |
|---|---|---|
| Batas Auth vs Identity/Organization | Auth hanya akun, token, role (2.3) | Sesuai daftar service di timeline; mencegah Auth menjadi service "serba ada" |
| Nama endpoint | Ikut nama API lama bila fungsinya sama | Familiar bagi lead dan tim frontend |
| Format respons | Standar baru `success/message/data` (5.6) | Seragam untuk semua service berikutnya; API lama tidak konsisten |
| Verifikasi OTP | Publik (email + OTP), tanpa token | Token tidak diterbitkan sebelum email terverifikasi |
| Login mahasiswa/dosen pakai NIM/NIDN | Belum; email dulu, struktur siap `identifier` | Belum dibutuhkan di PMB |
| OTP via WhatsApp | Belum; email saja, tabel OTP punya kolom `channel` | Butuh provider berbayar |
| Login Google | Dimatikan (feature flag) | Tidak ada di use case PMB |
| Penyimpanan token di frontend | Token di body JSON | Sesuai permintaan tugas frontend |
| Role yang punya scope | Hanya `prodi` | Sesuai use case |
| Database | Satu instance Postgres, database terpisah per service | Hemat resource, tetap terisolasi |

Jika lead memberi arahan berbeda, ubah bagian terkait dan catat di `docs/DECISIONS.md`.


---

# BAGIAN C — PRD FASE 2: SELURUH SERVICE PMB + OCR KTP

> Rujukan "Bagian X" di dalam bagian ini = Bagian C nomor X. Rujukan ke "PRD.md" = Bagian B. Untuk pembayaran, **Bagian D (CR-001 Duitku) menggantikan** ketentuan mock-only di C.1 dan C.4.5.

## (Judul asli) PRD Fase 2 — Layanan Inti Pendaftaran s/d NIM

| | |
|---|---|
| **Produk** | Sistem Pendaftaran Mahasiswa Baru (PMB), bagian dari "Sistem Kampus" |
| **Lingkup** | Seluruh service setelah Auth & Gateway: Organization, Identity, Notification, Admission, Finance, Exam, Scoring, Selection, Student, NIM |
| **Prasyarat** | Fase 1 selesai: Auth Service + API Gateway (lulus e2e 43/43) |
| **Stack** | Sama dengan fase 1: Go, Gin, GORM + PostgreSQL 15, Redis 7, Docker Compose, + MinIO (penyimpanan file) |
| **Dokumen acuan** | `docs/flowchart-sistem-pmb.pdf`, `docs/use-case-sistem-pmb.pdf`, `docs/class-diagram-auth-service.md`, `PRD.md` (fase 1), `docs/DECISIONS.md` |
| **Status** | Draft v2 — ditambah fitur OCR KTP + peningkatan kualitas gambar (permintaan lead) |

---

### Perubahan v2
- Fitur baru: **scan KTP dengan OCR** saat pengisian biodata, dengan **peningkatan kualitas gambar berbasis AI** bila foto buram atau kurang jelas (Bagian 4.11, milestone F2-M2b).
- Service baru: **OCR Service** (Python), hanya internal.

## 0. Instruksi untuk AI Agent

1. PRD fase 1 (`PRD.md`) **tetap berlaku** untuk semua aturan umum: format respons (5.6), error code, keamanan, struktur folder, dan cara kerja per milestone. PRD ini hanya menambah dan tidak mengubahnya, kecuali disebut eksplisit.
2. Kerjakan milestone di **Bagian 9** secara berurutan. Satu milestone = satu service (atau satu bagian service). Di akhir setiap milestone: `go build`, `go vet`, `go test`, `docker compose up -d --build` dari database kosong, `scripts/e2e.sh` semua PASS, commit, lalu laporkan dan **tunggu persetujuan**.
3. **Jangan pernah mengubah perilaku Auth Service dan Gateway yang sudah lulus e2e**, kecuali untuk menambah route di `routes.yaml`, menambah endpoint internal yang diminta PRD ini, atau memperbaiki bug. Setiap perubahan pada kedua service itu wajib tetap meluluskan seluruh e2e fase 1.
4. Jika PRD ini ambigu, pilih solusi paling sederhana yang konsisten dengan pola fase 1, lalu catat di `docs/DECISIONS.md`. Pertanyaan terbuka yang sudah diberi default ada di Bagian 11; gunakan default-nya, jangan menunggu jawaban.
5. Semua nama tabel, field JSON, dan identifier kode dalam bahasa Inggris. Pesan untuk user dalam bahasa Indonesia.

---

## 1. Tujuan

Menjadikan seluruh alur PMB di flowchart dapat dijalankan end-to-end lewat API Gateway:

> Registrasi → pilih gelombang, jalur, prodi → bayar pendaftaran → isi biodata & upload dokumen → verifikasi dokumen → kartu ujian → seleksi (tes/wawancara/berkas) → pengumuman → bayar daftar ulang (lunas/cicilan) → daftar ulang → NIM & akun mahasiswa.

Beserta sisi internal: admin PMB, keuangan, dosen, prodi, pimpinan, dan superadmin sesuai use case diagram.

### Non-tujuan fase 2
- Integrasi payment gateway **produksi** (Midtrans/Xendit/bank). Fase ini memakai adapter **mock** di balik interface, adapter nyata menyusul.
- Pengiriman WhatsApp nyata (adapter disiapkan, driver default `log`).
- Ujian online berbasis soal (CBT). Exam service hanya mengelola jadwal, peserta, kartu, dan kehadiran; nilai diinput dosen.
- Sistem akademik setelah mahasiswa aktif (KRS, perkuliahan).
- Frontend.

---

## 2. Peta Service & Kepemilikan Data

Prinsip dari fase 1 tetap: **satu service memiliki datanya sendiri**, satu database per service di satu instance Postgres, tidak ada service yang membaca database service lain. `user_id` (UUID dari Auth) adalah ID global orang.

| Service | Prefix gateway | Pemilik data | Keterangan |
|---|---|---|---|
| Auth *(fase 1)* | `/api/v1/auth` | Akun, token, role, scope | Tambahan kecil: lihat 4.10 |
| Gateway *(fase 1)* | — | — | Tambah route per service |
| **Organization** | `/api/v1/organization` | Fakultas, program studi | Master data kampus, dibaca banyak service |
| **Identity** | `/api/v1/identity` | Biodata orang, alamat, orang tua/wali, riwayat pendidikan, foto profil | Data personal dipisah dari Auth (arahan lead) |
| **Notification** | *(internal saja)* | Template & log notifikasi | Tambahan dari tim, lihat 4.3 |
| **Admission** | `/api/v1/admission` | Gelombang, jalur, kuota, persyaratan dokumen, **pendaftaran (application)**, dokumen & verifikasinya | Inti PMB |
| **Finance** | `/api/v1/finance` | Skema biaya, tagihan, pembayaran, cicilan, kwitansi | |
| **Exam** | `/api/v1/exam` | Jadwal seleksi, sesi, peserta, dosen penguji, kartu ujian, kehadiran | |
| **Scoring** | `/api/v1/scoring` | Komponen & bobot penilaian, nilai, rekap, ranking | |
| **Selection** | `/api/v1/selection` | Rekomendasi kelulusan, persetujuan, pengumuman | |
| **Student** | `/api/v1/student` | Daftar ulang, data mahasiswa baru | |
| **NIM** | `/api/v1/nim` | Format & urutan NIM, riwayat penerbitan | |
| **OCR** *(v2)* | *(internal saja)* | Tidak menyimpan data permanen; hanya job OCR sementara | Scan KTP + peningkatan kualitas gambar, lihat 4.11 |

**Aturan akses umum semua service baru:**
- Wajib middleware `pkg/gatewayauth` (verifikasi signature gateway) untuk semua route kecuali `/health*` dan `/internal/*`.
- Identitas user dibaca dari header `X-User-ID`, `X-User-Roles`, `X-User-Scopes` yang sudah diverifikasi gateway. Service **tidak** memvalidasi JWT ulang, kecuali memang dibutuhkan.
- Role guard kasar di gateway (`routes.yaml`), otorisasi detail di service. Contoh: role `prodi` hanya melihat data prodi pada scope-nya; `calon_mahasiswa` hanya melihat pendaftarannya sendiri.
- Endpoint internal `/internal/*` memakai `X-Internal-Token` dan tidak diekspos gateway.

---

## 3. Keterlacakan Use Case → Service

| UC | Use case | Service | Status |
|---|---|---|---|
| UC-01 | Login | Auth | Selesai (fase 1) |
| UC-02 | Registrasi Akun | Auth | Selesai |
| UC-03 | Verifikasi Email / OTP | Auth | Selesai |
| UC-04 | Pilih Gelombang & Jalur | Admission | Fase 2 |
| UC-05 | Pilih Program Studi | Admission (+ Organization) | Fase 2 |
| UC-06 | Isi Biodata | Identity | Fase 2 |
| UC-06a *(baru)* | Scan KTP (OCR + perbaikan kualitas foto), «extend» Isi Biodata | Identity + OCR | Fase 2 |
| UC-07 | Upload Dokumen Persyaratan | Admission | Fase 2 |
| UC-08 | Bayar Biaya Pendaftaran | Finance | Fase 2 |
| UC-09 | Cetak Kartu Ujian | Exam | Fase 2 |
| UC-10 | Lihat Pengumuman Hasil | Selection | Fase 2 |
| UC-11 | Bayar Daftar Ulang | Finance | Fase 2 |
| UC-12 | Daftar Ulang | Student | Fase 2 |
| UC-13 | Proses Pembayaran | Finance | Fase 2 |
| UC-14 | Upload Bukti Transfer | Finance | Fase 2 |
| UC-15 | Ajukan Cicilan | Finance | Fase 2 |
| UC-16 | Generate NIM | NIM | Fase 2 |
| UC-17 | Kirim Notifikasi | Notification | Fase 2 |
| UC-18 | Kelola Data Pendaftar | Admission | Fase 2 |
| UC-19 | Verifikasi Dokumen | Admission | Fase 2 |
| UC-20 | Kelola Jadwal Seleksi | Exam | Fase 2 |
| UC-21 | Assign Dosen Penguji | Exam | Fase 2 |
| UC-22 | Publikasikan Pengumuman | Selection | Fase 2 |
| UC-23 | Verifikasi Pembayaran | Finance | Fase 2 |
| UC-24 | Setujui Pengajuan Cicilan | Finance | Fase 2 |
| UC-25 | Lihat Laporan Keuangan | Finance | Fase 2 |
| UC-26 | Export Laporan | Finance, Scoring, Admission | Fase 2 |
| UC-27 | Input Nilai Seleksi | Scoring | Fase 2 |
| UC-28 | Lihat Rekap & Ranking | Scoring | Fase 2 |
| UC-29 | Rekomendasi Kelulusan | Selection | Fase 2 |
| UC-30 | Setujui Kelulusan | Selection | Fase 2 |
| UC-31 | Lihat Dashboard Statistik | Admission + Finance (endpoint statistik) | Fase 2 |
| UC-32 | Kelola Master Data | Organization (prodi), Admission (gelombang, jalur, kuota, dokumen), Finance (biaya) | Fase 2 |
| UC-33 | Kelola User & Role | Auth | Selesai |

---

## 4. Spesifikasi per Service

Setiap service mengikuti struktur `auth-service` (cmd/internal, repository, dto, migrations SQL, swagger, Dockerfile dengan BuildKit cache, `.env.example`). Skema di bawah adalah **minimum**; boleh ditambah kolom teknis (`created_at`, `updated_at`, `deleted_at`, `created_by`) di semua tabel.

### 4.1 Organization Service

**Tabel**
- `faculties`: `id`, `code` (unik, mis. `FT`), `name`, `is_active`.
- `study_programs`: `id`, `faculty_id`, `code` (unik, dipakai sebagai `scope_id` role prodi, mis. `TI`), `nim_code` (2 digit, untuk format NIM), `name`, `degree` (`D3`, `S1`, `S2`), `accreditation`, `is_active`.

**Endpoint**
| Method | Path | Akses |
|---|---|---|
| GET | `/faculties`, `/study-programs` (filter `faculty_code`, `is_active`) | Publik (dipakai form pendaftaran) |
| GET | `/study-programs/:code` | Publik |
| POST/PUT/PATCH | `/admin/faculties[/:id]`, `/admin/study-programs[/:id]` | `superadmin` |
| GET | `/internal/study-programs/:code`, `POST /internal/study-programs/batch` | Internal |

**Tambahan:** Auth Service mulai memvalidasi `scope_id` role `prodi` lewat `GET /internal/study-programs/:code` saat superadmin mengatur role (lihat 4.10).

Seeder development: 2 fakultas dan 4 prodi contoh.

### 4.2 Identity Service

Data personal, dipisah dari Auth sesuai arahan lead.

**Tabel**
- `persons`: `user_id` (PK, UUID dari Auth), `full_name`, `nik` (16 digit, unik, disimpan terenkripsi at-rest dengan AES-GCM dari env `IDENTITY_ENCRYPTION_KEY`, plus kolom `nik_hash` untuk pencarian unik), `birth_place`, `birth_date`, `gender`, `religion`, `nationality`, `phone`, `photo_object_key`.
- `addresses`: `user_id`, `type` (`ktp`, `domisili`), `street`, `rt`, `rw`, `village`, `district`, `city`, `province`, `postal_code`.
- `guardians`: `user_id`, `relation` (`ayah`, `ibu`, `wali`), `name`, `occupation`, `income_range`, `phone`.
- `education_histories`: `user_id`, `level` (`SMA`, `SMK`, `MA`, `D3`, …), `school_name`, `npsn`, `major`, `graduation_year`.

**Endpoint**
| Method | Path | Akses |
|---|---|---|
| GET/PUT | `/me/profile`, `/me/addresses`, `/me/guardians`, `/me/education` | User pemilik data |
| PUT | `/me/photo` (multipart, jpg/png maks 2 MB, disimpan di MinIO) | User |
| GET | `/me/completeness` → `{ "complete": bool, "missing": [..] }` | User |
| GET | `/admin/persons/:user_id` | `admin_pmb`, `superadmin` |
| GET | `/internal/persons/:user_id`, `/internal/persons/:user_id/completeness`, `POST /internal/persons/batch` | Internal |

Aturan kelengkapan biodata (dipakai Admission sebelum submit): profil, alamat KTP, minimal satu orang tua/wali, dan riwayat pendidikan terakhir wajib terisi.

NIK tidak pernah dikembalikan utuh ke role selain pemilik dan `admin_pmb`/`superadmin`; untuk role lain dimasking (`3205********0001`).

### 4.3 Notification Service *(tambahan tim)*

Satu tempat untuk kirim email/WA dan template, agar service lain tidak mengurus SMTP sendiri.

- Hanya endpoint internal: `POST /internal/notifications` body `{ "user_id" | "email", "channel": ["email","whatsapp"], "template": "document_verified", "data": {..}, "idempotency_key": "..." }`.
- Tabel `notification_logs`: status `queued`, `sent`, `failed`, `retries`, `error`.
- Pengiriman asinkron lewat antrean Redis (list/stream) + worker di service yang sama, retry 3 kali dengan backoff.
- Driver email: `smtp` / `log` (Mailpit di development). Driver WA: interface `WhatsAppSender` dengan driver `log` saja di fase ini.
- Template (HTML + teks, bahasa Indonesia): `payment_received`, `document_verified`, `document_revision`, `exam_card_ready`, `announcement_published`, `installment_reminder`, `reregistration_completed`, `nim_issued`.
- Auth Service **tidak** wajib dimigrasikan ke Notification Service di fase ini (sudah stabil). Catat sebagai pekerjaan lanjutan.

### 4.4 Admission Service

**Master data (UC-32)**
- `periods` (gelombang): `id`, `academic_year` (mis. `2026/2027`), `name` (mis. `Gelombang 1`), `registration_start`, `registration_end`, `exam_date_start`, `exam_date_end`, `announcement_date`, `reregistration_end`, `is_active`.
- `admission_paths` (jalur): `id`, `code` (`REGULER`, `PRESTASI`, `BEASISWA`, `PINDAHAN`), `name`, `description`, `selection_method` (`test`, `interview`, `document`, `test_interview`), `is_active`.
- `period_paths`: gelombang mana membuka jalur mana.
- `quotas`: `period_id`, `path_id`, `study_program_code`, `quota`.
- `document_requirements`: `path_id`, `code` (`IJAZAH_SKL`, `KTP`, `KK`, `PAS_FOTO`, `RAPOR`, `SERTIFIKAT_PRESTASI`, `TRANSKRIP`, `SURAT_PINDAH`), `name`, `is_required`, `allowed_types`, `max_size_mb`.

**Pendaftaran**
- `applications`: `id`, `registration_number` (format `PMB-{YYYY}-{period}-{urut 5 digit}`), `user_id`, `period_id`, `path_id`, `first_choice_program`, `second_choice_program`, `status`, `submitted_at`, `verified_at`, `verified_by`, `notes`.
- `application_documents`: `application_id`, `requirement_code`, `object_key`, `file_name`, `mime_type`, `size`, `status` (`uploaded`, `valid`, `invalid`), `review_note`, `reviewed_by`, `reviewed_at`.
- `application_status_logs`: riwayat perubahan status (siapa, kapan, dari → ke, catatan).

**Status pendaftaran (state machine, wajib divalidasi di service):**

```
draft ──(pilih gelombang/jalur/prodi, tagihan dibuat)──> awaiting_payment
awaiting_payment ──(Finance: tagihan pendaftaran lunas)──> form_open
form_open ──(biodata lengkap + dokumen wajib terunggah, submit)──> submitted
submitted ──(admin)──> verified | revision_required
revision_required ──(upload ulang + submit)──> submitted
verified ──(Exam: dijadwalkan)──> exam_scheduled
exam_scheduled ──(Selection: pengumuman terbit)──> passed | failed
passed ──(Student: daftar ulang selesai)──> reregistered
reregistered ──(NIM terbit)──> enrolled
```

Aturan: satu user hanya boleh punya **satu pendaftaran aktif per gelombang**. Perubahan prodi hanya boleh saat `draft`/`awaiting_payment`. Pendaftaran tidak bisa dibuat di luar rentang `registration_start`–`registration_end`.

**Endpoint utama**
| Method | Path | Akses |
|---|---|---|
| GET | `/periods/active`, `/periods/:id/paths`, `/paths/:id/requirements` | Publik |
| POST | `/applications` (pilih gelombang, jalur, prodi 1 & 2) | `calon_mahasiswa` |
| GET | `/applications/me` | `calon_mahasiswa` |
| PUT | `/applications/me/choices` | `calon_mahasiswa` (status `draft`/`awaiting_payment`) |
| POST | `/applications/me/documents` (multipart) | `calon_mahasiswa` |
| DELETE | `/applications/me/documents/:code` | `calon_mahasiswa` (sebelum submit) |
| POST | `/applications/me/submit` | `calon_mahasiswa` |
| GET | `/admin/applications` (filter status, gelombang, jalur, prodi, q; paginasi) | `admin_pmb`, `superadmin`, `prodi` (hanya prodinya), `pimpinan` (read-only) |
| GET | `/admin/applications/:id` (+ biodata dari Identity, status bayar dari Finance) | sama |
| GET | `/admin/applications/:id/documents/:code/url` (presigned URL 5 menit) | `admin_pmb`, `superadmin` |
| POST | `/admin/applications/:id/verify` body per dokumen `valid`/`invalid` + catatan | `admin_pmb` |
| GET | `/admin/applications/export` (xlsx) | `admin_pmb`, `superadmin`, `pimpinan` |
| CRUD | `/admin/periods`, `/admin/paths`, `/admin/quotas`, `/admin/requirements` | `superadmin` (baca juga `admin_pmb`) |
| GET | `/admin/stats` (jumlah per status, per prodi, per jalur, per gelombang; konversi pendaftar → daftar ulang) | `pimpinan`, `admin_pmb`, `superadmin` |
| GET/POST | `/internal/applications/:id`, `/internal/applications/:id/status` | Internal |

Setiap perubahan status penting memanggil Notification Service (verifikasi, revisi).

**Penyimpanan dokumen:** MinIO (S3-compatible) di docker-compose. Bucket privat per service (`admission-documents`). File tidak pernah diakses publik; hanya lewat presigned URL berumur pendek. Validasi MIME dari isi file (bukan ekstensi), ukuran maks per requirement (default 2 MB, PDF/JPG/PNG).

### 4.5 Finance Service

**Tabel**
- `fee_schemes`: `period_id`, `path_id`, `study_program_code` (nullable = berlaku semua prodi), `fee_type` (`registration`, `reregistration`), `amount` (integer rupiah, **jangan float**), `installment_allowed`, `max_installments`, `min_first_payment_percent`.
- `invoices`: `id`, `invoice_number` (`INV-{YYYYMMDD}-{urut}`), `user_id`, `application_id`, `fee_type`, `amount`, `paid_amount`, `status` (`unpaid`, `partially_paid`, `paid`, `cancelled`, `expired`), `due_date`, `payment_scheme` (`full`, `installment`).
- `installment_requests`: `invoice_id`, `requested_terms`, `reason`, `status` (`pending`, `approved`, `rejected`), `reviewed_by`, `review_note`.
- `installments`: `invoice_id`, `term_no`, `amount`, `due_date`, `status`, `paid_at`.
- `payments`: `id`, `invoice_id`, `installment_id` (nullable), `method` (`virtual_account`, `manual_transfer`), `provider`, `provider_ref`, `va_number`, `amount`, `status` (`pending`, `success`, `failed`, `expired`, `awaiting_verification`, `rejected`), `proof_object_key`, `verified_by`, `verified_at`, `paid_at`.
- `receipts`: `payment_id`, `receipt_number`, `pdf_object_key`.
- `payment_webhook_logs`: payload mentah + hasil verifikasi signature (untuk audit).

**Aturan**
- Tagihan pendaftaran dibuat otomatis saat Admission membuat pendaftaran (panggilan internal `POST /internal/invoices`). Pendaftaran selalu `full`.
- Tagihan daftar ulang dibuat saat Selection menerbitkan status `passed`.
- Cicilan hanya untuk `reregistration` dan hanya bila `installment_allowed`. Default: maks 4 termin, pembayaran pertama minimal 25%, jatuh tempo per 30 hari.
- **Payment provider** lewat interface `PaymentProvider` (`CreateVA`, `ParseWebhook`, `VerifySignature`). Fase ini: `MockProvider` + endpoint simulasi bayar khusus development `POST /dev/payments/:id/simulate-paid` (hanya aktif bila `APP_ENV=development`).
- Webhook provider: route publik `POST /webhooks/payments/:provider` di gateway (tanpa JWT), diverifikasi dengan signature provider, **idempoten** berdasarkan `provider_ref`.
- Transfer manual: user unggah bukti → status `awaiting_verification` → keuangan setujui/tolak.
- Saat tagihan pendaftaran lunas → panggil Admission `POST /internal/applications/:id/status` ke `form_open`. Saat daftar ulang lunas atau termin pertama dibayar → beri tahu Student Service bahwa daftar ulang boleh dilanjutkan.
- Semua perhitungan uang memakai integer; total `payments` sukses tidak boleh melebihi `amount` tagihan.
- Job terjadwal (goroutine ticker atau cron ringan di service): reminder termin H-3 dan H-1 via Notification; menandai tagihan `expired` sesuai `due_date`.

**Endpoint utama**
| Method | Path | Akses |
|---|---|---|
| GET | `/invoices/me`, `/invoices/me/:id` | pemilik |
| POST | `/invoices/me/:id/pay` body `{ "method": "virtual_account" }` → nomor VA | pemilik |
| POST | `/invoices/me/:id/manual-transfer` (multipart bukti) | pemilik |
| POST | `/invoices/me/:id/installment-request` | pemilik (tagihan daftar ulang) |
| GET | `/receipts/me/:payment_id` (PDF) | pemilik |
| GET | `/admin/payments?status=awaiting_verification` | `keuangan` |
| POST | `/admin/payments/:id/verify` (`approve`/`reject` + catatan) | `keuangan` |
| GET | `/admin/installment-requests`, `POST /admin/installment-requests/:id/review` | `keuangan` |
| CRUD | `/admin/fee-schemes` | `superadmin`, `keuangan` |
| GET | `/admin/reports/summary` (pemasukan per periode/jalur/prodi, tunggakan) | `keuangan`, `pimpinan`, `superadmin` |
| GET | `/admin/reports/export?format=xlsx` | `keuangan`, `pimpinan`, `superadmin` |
| POST | `/webhooks/payments/:provider` | Publik (signature provider) |
| POST | `/internal/invoices`, `GET /internal/invoices?application_id=` | Internal |

### 4.6 Exam Service

**Tabel**
- `exam_sessions`: `id`, `period_id`, `path_id`, `type` (`written_test`, `interview`), `study_program_code` (nullable), `start_at`, `end_at`, `location` atau `meeting_link`, `capacity`.
- `session_examiners`: `session_id`, `examiner_user_id` (role `dosen`).
- `participants`: `session_id`, `application_id`, `user_id`, `participant_number` (unik), `attendance` (`present`, `absent`, `null`).
- `exam_cards`: `participant_id`, `pdf_object_key`, `generated_at`.

**Aturan**
- Hanya pendaftaran berstatus `verified` yang bisa dijadwalkan. Setelah dijadwalkan, Exam memberi tahu Admission (`exam_scheduled`) dan Notification (`exam_card_ready`).
- Jalur dengan `selection_method = document` tidak perlu sesi ujian; langsung ke Scoring (penilaian berkas).
- Kartu ujian PDF berisi: nomor peserta, nama, foto (dari Identity), prodi pilihan, jadwal, lokasi/link, QR berisi nomor peserta.

**Endpoint utama**
| Method | Path | Akses |
|---|---|---|
| CRUD | `/admin/sessions` | `admin_pmb` |
| POST | `/admin/sessions/:id/examiners` | `admin_pmb` |
| POST | `/admin/sessions/:id/participants` (tambah manual atau auto-assign dari pendaftar `verified`) | `admin_pmb` |
| PATCH | `/admin/participants/:id/attendance` | `admin_pmb`, `dosen` (sesinya sendiri) |
| GET | `/me/schedule`, `/me/card` (PDF) | `calon_mahasiswa` |
| GET | `/examiner/sessions` (sesi yang ditugaskan) | `dosen` |

### 4.7 Scoring Service

**Tabel**
- `score_components`: `path_id`, `code` (`written_test`, `interview`, `report_card`, `achievement`), `name`, `weight_percent` (total per jalur = 100), `max_score`.
- `scores`: `application_id`, `component_code`, `score`, `input_by`, `input_at`, `note`.
- `score_summaries`: `application_id`, `final_score`, `rank_in_program`, `computed_at` (dihitung ulang saat nilai berubah).

**Aturan**
- Dosen hanya bisa input nilai untuk peserta di sesi yang ditugaskan kepadanya (cek ke Exam via internal API).
- Nilai tidak bisa diubah setelah Selection mengunci rekomendasi (flag `locked` per gelombang+prodi dari Selection).
- Ranking per prodi pilihan 1, per gelombang, per jalur.

**Endpoint utama**
| Method | Path | Akses |
|---|---|---|
| CRUD | `/admin/components` | `superadmin`, `admin_pmb` |
| GET | `/examiner/participants?session_id=` | `dosen` |
| PUT | `/examiner/scores/:application_id` | `dosen` |
| GET | `/program/recap?period_id=&path_id=` (rekap & ranking) | `prodi` (scope sendiri), `admin_pmb`, `pimpinan` |
| GET | `/program/recap/export` (xlsx) | sama |
| GET | `/internal/summaries?period_id=&program=` | Internal |

### 4.8 Selection Service

**Tabel**
- `recommendations`: `period_id`, `path_id`, `study_program_code`, `status` (`draft`, `submitted`, `approved`, `returned`), `submitted_by`, `approved_by`, `note`.
- `recommendation_items`: `recommendation_id`, `application_id`, `decision` (`pass`, `fail`, `waitlist`), `final_score`, `rank`.
- `announcements`: `period_id`, `title`, `published_at`, `published_by`.
- `results`: `application_id`, `user_id`, `decision`, `accepted_program`, `announced_at`.

**Alur (flowchart 4)**
1. Prodi melihat rekap (dari Scoring), menandai `pass`/`fail`/`waitlist` sesuai kuota (kuota dari Admission). Sistem menolak jumlah `pass` melebihi kuota.
2. Prodi submit rekomendasi → Scoring dikunci untuk kombinasi itu.
3. `pimpinan` menyetujui atau mengembalikan (dengan catatan) → bila dikembalikan, kembali ke prodi.
4. `admin_pmb` mempublikasikan pengumuman untuk rekomendasi yang `approved` → isi `results`, update status di Admission (`passed`/`failed`), minta Finance membuat tagihan daftar ulang untuk yang lulus, kirim notifikasi.
5. Hasil tidak terlihat oleh calon mahasiswa sebelum `published_at`.

**Endpoint utama**
| Method | Path | Akses |
|---|---|---|
| GET/PUT | `/program/recommendations` (per gelombang, jalur, prodi) | `prodi` |
| POST | `/program/recommendations/:id/submit` | `prodi` |
| GET | `/approval/recommendations`, `POST /approval/recommendations/:id/(approve\|return)` | `pimpinan` |
| POST | `/admin/announcements` (publikasi) | `admin_pmb` |
| GET | `/me/result` | `calon_mahasiswa` |

### 4.9 Student Service & NIM Service

**Student Service**
- `reregistrations`: `application_id`, `user_id`, `accepted_program`, `status` (`waiting_payment`, `in_progress`, `completed`), `completed_at`.
- `students`: `user_id`, `nim`, `study_program_code`, `entry_year`, `entry_path`, `status` (`active`).
- Syarat daftar ulang: hasil `passed`, tagihan daftar ulang lunas **atau** termin pertama cicilan terbayar, data tambahan terisi (ukuran almamater, konfirmasi data diri, pernyataan persetujuan). Setelah `completed` → panggil NIM Service.
- Endpoint: `GET /me/reregistration`, `PUT /me/reregistration` (data tambahan), `POST /me/reregistration/complete` (`calon_mahasiswa`); `GET /admin/students`, export xlsx (`admin_pmb`, `prodi` scope, `pimpinan`).

**NIM Service**
- `nim_formats`: template dapat dikonfigurasi. **Default:** `{YY}{nim_code_prodi}{jalur_digit}{urut 4}` → contoh `2655 1 0001` = `265510001` (YY = 2 digit tahun masuk, `nim_code` dari Organization, `jalur_digit` 1=reguler, 2=prestasi, 3=beasiswa, 4=pindahan).
- `nim_sequences`: `entry_year`, `study_program_code`, `last_number`, diperbarui dalam transaksi dengan `SELECT … FOR UPDATE` agar **tidak ada NIM ganda** walau request bersamaan.
- `nim_issuances`: riwayat penerbitan.
- Alur: `POST /internal/nim/issue` (dipanggil Student) → generate NIM → Student menyimpan ke `students` → Auth `POST /internal/users/:id/roles` menambah role `mahasiswa` → Admission status `enrolled` → notifikasi `nim_issued`.
- Endpoint admin: `GET /admin/issuances`, CRUD `/admin/formats` (`superadmin`).
- Idempoten: memanggil `issue` dua kali untuk `application_id` yang sama mengembalikan NIM yang sama.

### 4.10 Perubahan Kecil pada Auth & Gateway

- **Auth:** validasi `scope_id` role `prodi` ke Organization Service saat superadmin mengatur role. Jika Organization tidak tersedia, tolak dengan `UPSTREAM_UNAVAILABLE` (jangan simpan scope yang tidak tervalidasi).
- **Auth:** endpoint `POST /internal/users/:id/roles` dipakai NIM flow (sudah ada di fase 1; pastikan idempoten).
- **Gateway:** tambah route semua service di `routes.yaml` sesuai tabel akses di atas; tambah route publik webhook `POST /api/v1/finance/webhooks/payments/:provider`; body limit 5 MB untuk route upload (Identity foto, Admission dokumen, Finance bukti transfer).

---

### 4.11 OCR KTP & Peningkatan Kualitas Gambar *(v2)*

**Tujuan:** calon mahasiswa cukup memfoto atau mengunggah scan KTP, lalu data biodata (NIK, nama, tempat/tanggal lahir, jenis kelamin, alamat, RT/RW, kelurahan/desa, kecamatan, agama, kewarganegaraan) terisi otomatis. Bila foto buram, gelap, miring, atau beresolusi rendah, sistem memperbaiki kualitasnya dulu sebelum dibaca. Jika tetap tidak terbaca, user diminta foto ulang dengan petunjuk yang jelas.

**Prinsip yang tidak boleh dilanggar**
1. **Hasil OCR hanya saran, bukan data final.** Semua field tampil sebagai isian yang **wajib dikonfirmasi/diedit user** sebelum disimpan. Tidak ada auto-save.
2. **Peningkatan kualitas tidak boleh "mengarang" huruf/angka.** Dilarang memakai model generatif (mis. GAN super-resolution) yang bisa menciptakan karakter yang tidak ada. Hanya teknik yang mempertahankan isi: koreksi perspektif, deskew, denoise, sharpening, kontras adaptif (CLAHE), binarisasi, dan super-resolution ringan non-generatif (mis. OpenCV DNN FSRCNN/ESPCN).
3. **Gambar asli adalah dokumen resmi.** Gambar hasil perbaikan hanya dipakai untuk OCR dan tidak menggantikan file asli yang diverifikasi admin.
4. **Data KTP adalah data pribadi (UU No. 27/2022 tentang Pelindungan Data Pribadi).** Proses default berjalan **lokal di server sendiri**. OCR Service tidak menyimpan gambar maupun hasil secara permanen, dan log tidak boleh memuat NIK atau gambar. Penyedia AI/OCR cloud hanya boleh diaktifkan lewat konfigurasi, dengan persetujuan eksplisit user dan persetujuan lead (default: **mati**).

**Arsitektur**
- **OCR Service** ditulis dengan **Python 3.12 + FastAPI**, karena ekosistem OCR/computer vision (OpenCV, PaddleOCR) jauh lebih matang di Python. Ini satu-satunya service non-Go; tetap mengikuti kontrak fase 1: format respons 5.6, `X-Internal-Token`, `X-Request-ID`, healthcheck, Dockerfile non-root, `.env.example`.
- **Tidak diekspos gateway.** Hanya dipanggil Identity Service lewat `/internal/*`.
- Engine lewat interface `OcrEngine` (default `PaddleOCR`, cadangan `Tesseract` dengan bahasa `ind`) dan `ImageEnhancer`. Adapter cloud (`CloudVisionEngine`) disiapkan tetapi dimatikan (`OCR_CLOUD_ENABLED=false`).
- Pemrosesan **asinkron**: job masuk antrean Redis, worker memproses, hasil disimpan sementara di Redis dengan TTL 15 menit lalu terhapus otomatis.

**Pipeline**
1. **Validasi input:** JPG/PNG/PDF (halaman 1), maks 5 MB, resolusi minimal 640 px sisi terpendek.
2. **Deteksi & crop kartu:** temukan kontur kartu, koreksi perspektif, luruskan kemiringan, normalisasi orientasi.
3. **Cek kualitas** → skor 0–100 dari: ketajaman (variansi Laplacian), kecerahan & kontras, silau/pantulan, resolusi efektif, dan persentase kartu di frame.
4. **Perbaikan bila skor < ambang (default 70):** denoise, CLAHE, sharpening, koreksi pencahayaan, reduksi silau, dan super-resolution non-generatif bila resolusi rendah. Hitung ulang skor.
5. **OCR** pada gambar terbaik (asli atau hasil perbaikan), per region field KTP.
6. **Parsing & validasi:**
   - NIK 16 digit; koreksi salah baca yang umum hanya pada posisi angka (`O→0`, `I/l→1`, `B→8`, `S→5`).
   - Konsistensi NIK: digit 7–12 = tanggal lahir DDMMYY (perempuan: tanggal + 40) harus cocok dengan tanggal lahir dan jenis kelamin hasil OCR; kode wilayah 6 digit awal harus valid (tabel kode wilayah Kemendagri di-seed ke OCR Service).
   - Normalisasi: huruf kapital, format tanggal `YYYY-MM-DD`, jenis kelamin → `L`/`P`.
7. **Hasil** per field: `value`, `confidence` (0–1), `needs_review` (true bila confidence < 0.85 atau gagal validasi konsistensi).

**Respons jika gagal:** status `retake_required` dengan alasan spesifik yang bisa ditampilkan frontend: `BLURRY` ("Foto buram, pegang HP lebih stabil"), `TOO_DARK`, `GLARE` ("Ada pantulan cahaya, hindari lampu langsung"), `CARD_NOT_FOUND` ("Pastikan seluruh KTP terlihat dalam bingkai"), `LOW_RESOLUTION`, `NOT_KTP`.

**Endpoint**

OCR Service (internal):
| Method | Path | Keterangan |
|---|---|---|
| POST | `/internal/ocr/ktp` (multipart) | Buat job → `{ "job_id": "..." }` |
| GET | `/internal/ocr/jobs/:id` | `processing` / `completed` / `retake_required` / `failed` + hasil |
| GET | `/health`, `/health/ready` | |

Identity Service (lewat gateway, role `calon_mahasiswa`):
| Method | Path | Keterangan |
|---|---|---|
| POST | `/me/ktp-scan` (multipart, maks 5 MB) | Simpan gambar asli ke MinIO (bucket privat `identity-ktp`), kirim ke OCR → `{ "scan_id": "..." }` |
| GET | `/me/ktp-scan/:scan_id` | Status + field saran + `quality_score` + `enhanced` (bool) + alasan bila `retake_required` |
| POST | `/me/ktp-scan/:scan_id/confirm` | User mengirim field yang **sudah dicek/diedit** → disimpan ke `persons`/`addresses` |

Identity mencatat tabel `ktp_scans`: `id`, `user_id`, `object_key` (gambar asli), `status`, `quality_score`, `enhanced`, `field_confidences` (jsonb, **tanpa nilai NIK**), `confirmed_at`, `fields_edited_by_user` (daftar field yang diubah user dari saran OCR). Nilai hasil OCR mentah tidak disimpan permanen.

**Dukungan verifikasi admin:** di detail pendaftar (Admission), tampilkan gambar KTP asli, `quality_score`, dan tanda bila user mengubah field penting (NIK, nama, tanggal lahir) dari hasil OCR, agar admin lebih teliti saat verifikasi dokumen. Ini membantu, bukan menggantikan, verifikasi manual.

**Rate limit:** maksimal 10 scan per user per jam.

**Konfigurasi (`.env.example` OCR Service):** `OCR_ENGINE=paddle`, `OCR_QUALITY_THRESHOLD=70`, `OCR_FIELD_CONFIDENCE_THRESHOLD=0.85`, `OCR_RESULT_TTL=15m`, `OCR_MAX_FILE_MB=5`, `OCR_CLOUD_ENABLED=false`, `INTERNAL_TOKEN`, `REDIS_ADDR`, `WORKER_CONCURRENCY=2`.

**Pengujian**
- **Dilarang memakai foto KTP orang sungguhan** di repo maupun test. Buat **KTP sintetis** (generator gambar dengan data palsu dan layout menyerupai KTP) di `services/ocr-service/tests/fixtures/`, plus varian buram (Gaussian/motion blur), gelap, silau, miring, dan resolusi rendah.
- Target akurasi pada dataset sintetis: NIK **exact match ≥ 95%** untuk gambar jernih dan **≥ 85%** untuk gambar buram ringan setelah perbaikan; field nama & tanggal lahir ≥ 90%.
- Gambar yang sangat buram/terpotong harus menghasilkan `retake_required`, **bukan** hasil yang salah dengan confidence tinggi.
- Unit test validasi NIK (tanggal lahir, jenis kelamin, kode wilayah) dan koreksi karakter.
- Laporan akurasi otomatis `make ocr-benchmark` → `docs/ocr-benchmark.md` (akurasi per field, sebelum vs sesudah perbaikan kualitas, waktu proses rata-rata).
- e2e: upload KTP sintetis → polling hasil → confirm → biodata tersimpan; upload gambar sangat buram → `retake_required`.

**Kinerja:** target ≤ 5 detik per KTP di CPU (tanpa GPU) untuk gambar biasa. Model diunduh saat build image, bukan saat runtime.

## 5. Komunikasi Antar Service

- **Sinkron:** HTTP internal ke `/internal/*` dengan header `X-Internal-Token` dan `X-Request-ID` diteruskan. Buat klien bersama di `pkg/internalclient` (timeout 5 detik, retry 2 kali untuk GET, circuit sederhana opsional).
- **Perubahan status lintas service** (mis. Finance → Admission saat lunas): panggilan internal yang **idempoten** (menyertakan `idempotency_key`), dengan tabel **outbox** di service pengirim + worker retry. Tujuannya: bila service tujuan sedang mati, perubahan tidak hilang.
- Event bus (Redis Streams/Kafka) **tidak** dipakai di fase ini; catat sebagai pengembangan berikutnya.
- Service tidak saling memanggil secara melingkar dalam satu request. Bila perlu data gabungan (mis. detail pendaftar = Admission + Identity + Finance), service pemilik halaman (Admission) yang mengumpulkan lewat internal API.

## 6. Infrastruktur

- `deploy/docker-compose.yml`: tambah MinIO (+ inisialisasi bucket), setiap service baru, database per service (dibuat oleh init script Postgres).
- Setiap service punya healthcheck; gateway `depends_on` semua service yang dirutekan.
- `Makefile`: target per service (`make run-admission`, `make test-finance`, dst.) dan `make e2e`.
- Seeder development lintas service: 1 gelombang aktif, 4 jalur, kuota, persyaratan dokumen, skema biaya, komponen nilai, akun contoh untuk setiap role (password di README).

## 7. Testing

- Unit test per service (target coverage `service` ≥ 70%), termasuk: state machine pendaftaran (transisi ilegal ditolak), kuota, perhitungan cicilan (integer, jumlah termin = total), idempotensi webhook, NIM tanpa duplikasi (test konkurensi), otorisasi scope prodi, akses data milik sendiri.
- `scripts/e2e.sh` diperluas bertahap per milestone hingga menjalankan **seluruh flowchart 1** lewat gateway:
  register → verify → buat pendaftaran → bayar (simulate-paid) → isi biodata → upload dokumen → submit → admin verifikasi (satu kali revisi dulu) → jadwal & kartu → dosen input nilai → prodi rekomendasi → pimpinan setujui → admin publikasi → cek hasil → ajukan cicilan → keuangan setujui → bayar termin 1 → daftar ulang → NIM terbit → login ulang dan token memuat role `mahasiswa`.
- Skenario negatif minimal per service: akses data user lain (403/404), role salah (403), prodi di luar scope (403), transisi status ilegal (409 `INVALID_STATE`), kuota penuh (409 `QUOTA_EXCEEDED`), file bukan PDF/JPG/PNG atau terlalu besar (422), webhook signature salah (401), bypass gateway (401).
- Error code baru dicatat di `docs/API.md`: `INVALID_STATE`, `QUOTA_EXCEEDED`, `PERIOD_CLOSED`, `DUPLICATE_APPLICATION`, `DOCUMENT_REQUIRED`, `BIODATA_INCOMPLETE`, `PAYMENT_REQUIRED`, `INSTALLMENT_NOT_ALLOWED`, `SCORES_LOCKED`, `RESULT_NOT_PUBLISHED`, `FILE_TYPE_NOT_ALLOWED`, `FILE_TOO_LARGE`, `WEBHOOK_SIGNATURE_INVALID`.

## 8. Dokumentasi per Service

Setiap milestone wajib menghasilkan:
- Swagger service tersebut.
- Bagian baru di `docs/API.md` (endpoint, contoh request/response, error code).
- Folder baru di Postman collection.
- `docs/class-diagram-<service>.md` (Mermaid, dibuat dari kode aktual, pola yang sama dengan auth).
- ERD Mermaid di `docs/erd-<service>.md`.
- Update `docs/DECISIONS.md` dan tabel keterlacakan (Bagian 3) dengan status terbaru.

## 9. Milestone

| Milestone | Isi | Bukti selesai |
|---|---|---|
| **F2-M0** Fondasi bersama | `pkg/internalclient`, `pkg/storage` (MinIO), `pkg/pagination`, `pkg/outbox`, template service baru, MinIO & init DB di compose | build/test lulus, MinIO bisa upload/download lewat test |
| **F2-M1** Organization | 4.1 + validasi scope di Auth (4.10) | superadmin membuat prodi; atur role prodi dengan scope tak dikenal ditolak |
| **F2-M2** Identity | 4.2 | calon mahasiswa mengisi biodata; `completeness` benar; NIK terenkripsi di DB |
| **F2-M2b** OCR KTP *(v2)* | 4.11: OCR Service (Python) + endpoint scan di Identity | KTP sintetis terbaca otomatis, gambar buram diperbaiki atau diminta foto ulang, benchmark akurasi memenuhi target |
| **F2-M3** Notification | 4.3 | notifikasi masuk Mailpit, retry saat SMTP mati |
| **F2-M4** Admission bagian 1 | Master data + buat pendaftaran + state machine awal | pendaftaran dibuat, kuota & periode tervalidasi |
| **F2-M5** Finance bagian 1 | Skema biaya, tagihan, VA mock, webhook, transfer manual, verifikasi, kwitansi | bayar pendaftaran → Admission `form_open` |
| **F2-M6** Admission bagian 2 | Upload dokumen, submit, verifikasi & revisi, export, statistik | alur sampai `verified` + notifikasi |
| **F2-M7** Exam | 4.6 | jadwal, dosen, peserta, kartu PDF |
| **F2-M8** Scoring | 4.7 | dosen input nilai, rekap & ranking prodi |
| **F2-M9** Selection | 4.8 | rekomendasi → persetujuan → pengumuman → tagihan daftar ulang |
| **F2-M10** Finance bagian 2 | Cicilan, reminder, laporan & export | ajukan & setujui cicilan, bayar termin |
| **F2-M11** Student + NIM | 4.9 | daftar ulang → NIM unik → role `mahasiswa` |
| **F2-M12** Integrasi akhir | e2e flowchart 1 penuh, dashboard statistik pimpinan, Postman & dokumentasi lengkap | `e2e.sh` seluruh alur PASS dari database kosong |

### 9.1 Pemetaan ke Timeline Tim

Pekerjaan dieksekusi **bertahap mengikuti baris timeline** (sheet Timeline Development IT). Satu tahap = satu atau dua milestone di atas.

| Tahap | Baris timeline | Milestone | Rencana tanggal |
|---|---|---|---|
| 1 | Organization Service | F2-M0 + F2-M1 | 22–25 Sep 2026 |
| 2 | Identity Service | F2-M2 + F2-M2b (OCR KTP) | 26 Sep – 6 Okt 2026 |
| 3 | Admission Service (1) | F2-M3 (Notification) + F2-M4 | 7–13 Okt 2026 |
| 4 | Finance Service (bagian 1) | F2-M5 | 14–20 Okt 2026 |
| 5 | Admission Service (2) | F2-M6 | 21–26 Okt 2026 |
| 6 | Exam Service | F2-M7 | 27 Okt – 2 Nov 2026 |
| 7 | Scoring Service | F2-M8 | 3–6 Nov 2026 |
| 8 | Selection Service | F2-M9 | 7–12 Nov 2026 |
| 9 | Finance Service (bagian 2) | F2-M10 | 13–18 Nov 2026 |
| 10 | Student Service | F2-M11 (bagian Student) | 19–21 Nov 2026 |
| 11 | NIM Service | F2-M11 (bagian NIM) + F2-M12 | 22–30 Nov 2026 |

Tambahan atau perubahan dari lead dimasukkan sebagai **Change Request** (lihat `docs/CHANGE_REQUESTS.md`) dan dikerjakan di tahap yang paling relevan tanpa merombak tahap yang sudah selesai.

Urutan boleh disesuaikan dengan sprint di timeline tim; jika berubah, catat di `docs/DECISIONS.md`. Prinsip yang tidak boleh dilanggar: service yang menjadi ketergantungan dikerjakan lebih dulu.

## 10. Definition of Done Fase 2

- [ ] Seluruh use case UC-01 s/d UC-33 berstatus **Selesai** di tabel keterlacakan.
- [ ] Alur flowchart 1–5 dapat dijalankan end-to-end lewat gateway dan tercakup di `scripts/e2e.sh`.
- [ ] Setiap service: signature gateway wajib, otorisasi per role & scope, test negatif lulus.
- [ ] Tidak ada service yang membaca database service lain.
- [ ] Uang disimpan sebagai integer; NIM dijamin unik; webhook idempoten.
- [ ] Dokumen dan foto hanya dapat diakses lewat presigned URL berumur pendek.
- [ ] Dokumentasi, Postman, class diagram, dan ERD setiap service tersedia dan sesuai kode.
- [ ] `docker compose up -d --build` dari clone bersih + `scripts/e2e.sh` semua PASS.
- [ ] OCR KTP memenuhi target akurasi di `docs/ocr-benchmark.md`, tidak menyimpan gambar/hasil secara permanen di OCR Service, dan tidak mengirim data ke pihak ketiga secara default.

## 11. Pertanyaan Terbuka (dipakai default dulu)

| Pertanyaan | Default sementara |
|---|---|
| Metode seleksi per jalur | Reguler: tes tulis + wawancara; Prestasi: berkas + wawancara; Beasiswa: tes tulis + wawancara; Pindahan: berkas. Dapat diubah via master jalur & komponen nilai |
| Aturan cicilan | Hanya daftar ulang, maks 4 termin, minimal 25% di termin pertama, jatuh tempo per 30 hari, tanpa denda |
| Vendor payment gateway | Mock provider; interface siap untuk Midtrans/Xendit |
| Dokumen wajib per jalur | Semua: KTP, KK, pas foto, ijazah/SKL. Reguler & Beasiswa: + rapor. Prestasi: + sertifikat prestasi. Pindahan: + transkrip & surat pindah |
| Siapa yang menyetujui kelulusan | Role `pimpinan` |
| Format & sumber NIM | Dibuat NIM Service dengan format default 4.9, dapat diubah superadmin |
| WhatsApp | Adapter `log`, belum mengirim sungguhan |
| Pembayaran pendaftaran bisa refund? | Tidak ada refund di sistem; dicatat manual oleh keuangan |
| OCR KTP: boleh memakai layanan AI/OCR cloud? | Tidak; proses lokal (PaddleOCR + OpenCV). Adapter cloud disiapkan tapi mati, perlu persetujuan lead + persetujuan user |
| OCR KTP: wajib atau opsional? | Opsional; user tetap bisa isi biodata manual |
| Server punya GPU? | Diasumsikan tidak; pipeline dioptimalkan untuk CPU |

---

# BAGIAN D — CHANGE REQUEST

Semua CR juga dicatat di `docs/CHANGE_REQUESTS.md` (nomor, tanggal, isi, service/tahap terdampak, status).

## CR-001 — Payment gateway memakai Duitku (24 Sep 2026, dari lead)

**Keputusan lead:** "Yang sudah ada kita pakai duitku.com, coba pelajari buat payment gatewaynya."

**Dampak:** Finance Service (Bagian C.4.5). Bagian C yang menyebut "adapter mock di fase ini, payment gateway produksi menyusul" **diganti** oleh CR ini: adapter **Duitku** wajib diimplementasikan sekarang (mode sandbox), MockProvider tetap ada untuk test otomatis.

### D.1 Ringkasan API Duitku (Duitku POP / createInvoice)

Sumber: dokumentasi resmi `https://docs.duitku.com/pop/en`. **Agent wajib membuka dokumentasi resmi untuk memastikan detail sebelum implementasi**; jika ada perbedaan dengan ringkasan ini, dokumentasi resmi yang berlaku dan catat di `docs/DECISIONS.md`.

**Create Invoice**
- `POST https://api-sandbox.duitku.com/api/merchant/createInvoice` (sandbox) / `https://api-prod.duitku.com/api/merchant/createInvoice` (production), `Content-Type: application/json`.
- Header:
  - `x-duitku-merchantcode`: merchant code
  - `x-duitku-timestamp`: UNIX timestamp **milidetik** (zona Jakarta)
  - `x-duitku-signature`: `HMAC_SHA256(merchantCode + timestamp, apiKey)`, hex lowercase. (Sejak Apr 2026 Duitku memakai HMAC; md5/sha256 biasa sudah usang.)
- Body utama: `paymentAmount` (integer, tanpa desimal), `merchantOrderId` (string ≤ 50, **unik**), `productDetails` (≤ 255), `email`, `phoneNumber` (opsional), `customerVaName` (≤ 20 karakter — potong nama bila lebih), `callbackUrl`, `returnUrl`, `expiryPeriod` (menit), `paymentMethod` (opsional; kosong = user memilih di halaman Duitku), `itemDetails` (opsional, total harus sama dengan `paymentAmount`).
- Respons: `merchantCode`, `reference` (**simpan** sebagai `provider_ref`), `paymentUrl`, `vaNumber` (bila ada), `amount`, `statusCode` (`00` = sukses), `statusMessage`.
- Error: 400 (request salah; atau `merchantOrderId` sama dengan nominal berbeda), 401, 409 (`merchantOrderId` sama masih diproses), 500.

**Callback (webhook)**
- Duitku mengirim `HTTP POST`, `Content-Type: application/x-www-form-urlencoded` ke `callbackUrl`.
- Field penting: `merchantCode`, `amount`, `merchantOrderId`, `paymentCode`, `resultCode` (`00` sukses, `01`/`02` gagal), `reference`, `signature`, `publisherOrderId`, `settlementDate`.
- Verifikasi: `signature == HMAC_SHA256(merchantCode + amount + merchantOrderId, apiKey)` dengan perbandingan constant-time. Juga cocokkan `merchantCode` dengan konfigurasi dan `amount` dengan nominal pembayaran yang tersimpan.
- **Redirect** ke `returnUrl` dengan query `merchantOrderId`, `resultCode`, `reference` — **tidak boleh** dijadikan dasar status lunas (bisa dipalsukan); status hanya dari callback terverifikasi atau cek status ke API Duitku.

**Kode metode pembayaran (contoh)**: `BC` BCA VA, `M2` Mandiri VA, `I1` BNI VA, `BR` BRIVA, `BT` Permata VA, `B1` CIMB VA, `A1` ATM Bersama, `BV` BSI VA, `DM` Danamon VA, `FT` Alfamart/Pegadaian/Pos, `IR` Indomaret, `OV` OVO, `DA` DANA, `SP` QRIS ShopeePay, `NQ` QRIS Nobu, `VC` kartu kredit. Default sistem: **kosongkan `paymentMethod`** agar user memilih di halaman Duitku, kecuali frontend mengirim pilihan.

### D.2 Desain di Finance Service

- `PaymentProvider` interface (sudah ada di desain Bagian C.4.5): `CreatePayment`, `ParseWebhook`, `VerifySignature`, dan tambahan `CheckStatus` (rekonsiliasi; implementasikan dengan endpoint cek status transaksi Duitku setelah memverifikasi spesifikasinya di dokumentasi resmi).
- Implementasi: `DuitkuProvider` (baru) dan `MockProvider` (tetap). Dipilih lewat env `PAYMENT_PROVIDER=duitku|mock`.
- `merchantOrderId` = ID pembayaran internal (`payments.id` atau nomor unik turunan) — **bukan** nomor invoice, karena satu invoice cicilan bisa punya banyak pembayaran. Simpan `reference`, `paymentUrl`, `vaNumber`, `paymentCode`, `publisherOrderId`.
- Idempotensi:
  - Membuat pembayaran untuk invoice/termin yang masih punya pembayaran `pending` dan belum kedaluwarsa → kembalikan pembayaran yang sama (paymentUrl lama), jangan buat baru.
  - Respons 409 dari Duitku → tunggu/ambil ulang, jangan buat `merchantOrderId` baru secara diam-diam.
  - Callback yang sama datang berkali-kali → hanya diproses sekali (kunci: `reference`/`merchantOrderId` + status final).
- Callback route publik di gateway: `POST /api/v1/finance/webhooks/payments/duitku` (tanpa JWT, tanpa signature gateway di sisi publik; gateway tetap meneruskan dengan signature gateway ke finance-service). Body limit kecil (64 KB). Rate limit terpisah.
- Semua payload callback mentah disimpan di `payment_webhook_logs` beserta hasil verifikasi (tanpa API key).
- Respons callback: HTTP 200 bila diterima & valid (termasuk duplikat); 400/401 bila signature/parameter salah.
- Job rekonsiliasi: setiap 10 menit, cek status pembayaran `pending` yang umurnya > 15 menit lewat `CheckStatus`, untuk mengatasi callback yang gagal terkirim.
- Endpoint `POST /dev/payments/:id/simulate-paid` tetap ada **hanya** saat `APP_ENV=development` dan `PAYMENT_PROVIDER=mock`.

### D.3 Konfigurasi

```
PAYMENT_PROVIDER=mock            # mock | duitku
DUITKU_ENV=sandbox               # sandbox | production
DUITKU_MERCHANT_CODE=
DUITKU_API_KEY=
DUITKU_EXPIRY_MINUTES=1440
DUITKU_CALLBACK_URL=             # URL publik ke gateway: https://<domain>/api/v1/finance/webhooks/payments/duitku
DUITKU_RETURN_URL=               # halaman frontend setelah bayar
DUITKU_HTTP_TIMEOUT=15s
```

- URL API dipilih otomatis dari `DUITKU_ENV`. Service gagal start bila `PAYMENT_PROVIDER=duitku` tetapi merchant code/API key kosong.
- **Development di laptop:** Duitku tidak bisa memanggil `localhost`. Untuk uji callback sandbox sungguhan, pakai tunnel (mis. `cloudflared tunnel --url http://localhost:8000` atau ngrok) lalu isi `DUITKU_CALLBACK_URL`. Tanpa tunnel, andalkan job rekonsiliasi `CheckStatus`. Dokumentasikan keduanya di README.
- API key tidak boleh muncul di log, respons, maupun repo.

### D.4 Pengujian CR-001

- Unit test: pembuatan signature header (vektor uji dengan nilai tetap), verifikasi signature callback (valid/invalid/merchantCode salah/amount tidak cocok), parsing form callback, pemetaan `resultCode`, penanganan 409 & 400, idempotensi callback, pemotongan `customerVaName` ≤ 20.
- Test HTTP dengan server tiruan (`httptest`) yang meniru respons Duitku — **tidak** memanggil Duitku sungguhan di test otomatis.
- `scripts/e2e.sh` tetap memakai `PAYMENT_PROVIDER=mock`.
- Script manual `scripts/duitku-sandbox-check.sh` (opsional dijalankan manusia): membuat invoice sandbox sungguhan dan mencetak `paymentUrl` untuk dibuka di browser. Hanya berjalan bila env Duitku terisi.
- Checklist manual (untuk Diar): daftar merchant sandbox di dashboard Duitku → isi env → buat pembayaran lewat Postman → buka `paymentUrl` → bayar dengan simulator VA sandbox Duitku → status di sistem menjadi lunas (via callback lewat tunnel atau via rekonsiliasi).

---

# BAGIAN E — KEPUTUSAN TAMBAHAN & MASALAH YANG DIKETAHUI

## E.1 Keputusan yang sudah dikonfirmasi lead

| Keputusan | Sumber |
|---|---|
| Auth hanya akun, login, token, role. Data personal di Identity Service | Lead, 18 Sep |
| Format respons standar baru, dikoordinasikan dengan frontend | Lead, 18 Sep |
| Payment gateway: Duitku | Lead, 24 Sep (CR-001) |
| Tambahan fitur: OCR KTP + peningkatan kualitas gambar | Lead, 21 Sep (Bagian C.4.11) |
| Schema database & class diagram tiap service diminta untuk review | Lead, 22 Sep |

## E.2 Aturan teknis tambahan (berlaku untuk semua service)

1. **Setiap module Go** (`pkg/` dan tiap `services/*`) wajib `go mod tidy` bersih; `go.sum` di-commit. CI/lokal gagal bila `go mod tidy` mengubah file (`git diff --exit-code go.mod go.sum`).
2. **Satu versi Go** untuk semua `go.mod` dan semua Dockerfile (pakai versi yang sudah dipakai Auth/Gateway yang lulus; saat ini image `golang:1.26-alpine` — samakan `student-service`).
3. Dockerfile semua service Go memakai BuildKit cache mount untuk `/go/pkg/mod` dan `/root/.cache/go-build`, build context root repo, user non-root, tanpa `COPY .env`.
4. `.dockerignore` root mengabaikan `reference/`, `.git/`, `docs/*.pdf`, `postgres-data/`, `**/*.pem`, `**/.env`.
5. GORM logger tidak mencetak `ErrRecordNotFound` dan tidak mencetak SQL mentah kecuali `LOG_LEVEL=debug`.
6. Gateway hanya meneruskan satu `X-Request-Id` (tidak duplikat).
7. Setiap service baru wajib menolak request tanpa signature gateway (`GATEWAY_SIGNATURE_INVALID`) kecuali `/health*`, `/internal/*` (pakai `X-Internal-Token`), dan route yang secara eksplisit publik lewat gateway (mis. webhook Duitku tetap datang lewat gateway sehingga tetap bersignature gateway).
8. Port service selain gateway **tidak** di-expose ke host kecuali lewat profile/override development yang terdokumentasi.
9. Tidak ada service yang membaca database service lain.
10. Folder `reference/` tidak pernah di-commit.

## E.3 Pemetaan tahap ke timeline tim

| Tahap | Baris timeline | Isi |
|---|---|---|
| 1 | Organization Service | Fondasi bersama (C.9 F2-M0) + Organization (C.4.1) |
| 2 | Identity Service | Identity (C.4.2) + OCR KTP (C.4.11) |
| 3 | Admission Service (1) | Notification (C.4.3) + Admission bagian 1 |
| 4 | Finance Service (1) | Finance bagian 1 + **CR-001 Duitku** |
| 5 | Admission Service (2) | Admission bagian 2 |
| 6 | Exam Service | C.4.6 |
| 7 | Scoring Service | C.4.7 |
| 8 | Selection Service | C.4.8 |
| 9 | Finance Service (2) | Finance bagian 2 (cicilan, laporan) |
| 10 | Student Service | C.4.9 bagian Student |
| 11 | NIM Service | C.4.9 bagian NIM + integrasi akhir |

---
