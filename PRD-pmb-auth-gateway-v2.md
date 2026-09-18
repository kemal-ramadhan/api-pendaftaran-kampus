# PRD — Sistem PMB: Auth Service & API Gateway (Backend Fase 1)

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

## 0. Instruksi untuk AI Agent (baca dulu)

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
