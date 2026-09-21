# Kontrak API — Sistem PMB (Fase 1: Auth & Gateway)

Dokumen ini adalah spesifikasi integrasi resmi API Backend Sistem PMB untuk Tim Pengembang Frontend dan Service Hilir.
Semua komunikasi klien publik diarahkan melalui **API Gateway** pada:

```
Base URL: http://localhost:8000/api/v1
```

---

## 1. Standar Desain Respons & Error

### A. Format Respons Sukses
Semua respons sukses membungkus data dalam amplop JSON standar:
```json
{
  "success": true,
  "message": "Pesan deskriptif keberhasilan dalam Bahasa Indonesia",
  "data": {
    ...
  },
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 120
  }
}
```
*Catatan: Objek `meta` hanya disertakan pada endpoint yang memiliki pagination (seperti daftar pengguna).*

### B. Format Respons Error
Semua respons error membungkus informasi kesalahan secara seragam:
```json
{
  "success": false,
  "message": "Pesan ramah pengguna dalam Bahasa Indonesia",
  "error_code": "INVALID_CREDENTIALS",
  "errors": {
    "email": ["Format email tidak valid"],
    "password": ["Kata sandi minimal 8 karakter"]
  },
  "request_id": "c56a4180-65aa-42ec-a945-5fd21dec0538"
}
```
*Catatan: Objek `errors` berisi pemetaan nama field form ke pesan error validasi, dan hanya muncul pada HTTP `422 Unprocessable Entity`.*

---

## 2. Kamus Error Code Standar

| Error Code | HTTP Status | Keterangan & Tindakan Frontend |
|---|---|---|
| `VALIDATION_ERROR` | 422 | Input form tidak valid; petakan objek `errors` ke form field terkait. |
| `EMAIL_TAKEN` | 409 | Email sudah terdaftar dan terverifikasi; sarankan user untuk login. |
| `INVALID_CREDENTIALS` | 401 | Email atau kata sandi salah (pesan generik anti-enumerasi). |
| `EMAIL_NOT_VERIFIED` | 403 | Akun belum verifikasi OTP; arahkan pengguna ke layar verifikasi OTP. |
| `ACCOUNT_INACTIVE` | 403 | Akun dinonaktifkan administrator; tampilkan pesan hubungi admin. |
| `TOO_MANY_ATTEMPTS` | 429 | Gagal login 5x berturut-turut; akun terkunci selama 15 menit. |
| `OTP_INVALID` | 400 | Kode OTP salah; tampilkan sisa kesempatan percobaan. |
| `OTP_EXPIRED` | 400 | Kode OTP telah kedaluwarsa (> 10 menit); arahkan kirim ulang. |
| `OTP_LOCKED` | 429 | Gagal OTP 5x berturut-turut; wajib klik tombol kirim ulang OTP. |
| `OTP_RESEND_COOLDOWN` | 429 | Tombol kirim ulang diklik sebelum cooldown 60 detik selesai. |
| `RESET_TOKEN_INVALID` | 400 | Token reset password salah, kedaluwarsa (> 30 m), atau sudah dipakai. |
| `TOKEN_MISSING` | 401 | Header `Authorization: Bearer <token>` tidak disertakan. |
| `TOKEN_INVALID` | 401 | Format token rusak, algoritma salah, atau bukan `typ=access`. |
| `TOKEN_EXPIRED` | 401 | Access token kedaluwarsa; picu rotasi token via `/auth/refresh`. |
| `TOKEN_REVOKED` | 401 | Token berada di Redis denylist atau sesi dicabut (misal reuse terdeteksi). |
| `FORBIDDEN` | 403 | Pengguna tidak memiliki role/scope yang disyaratkan. |
| `NOT_FOUND` | 404 | Endpoint tidak ditemukan atau akses ke internal diblokir. |
| `RATE_LIMITED` | 429 | IP melebihi ambang batas request gateway. |
| `GATEWAY_SIGNATURE_INVALID` | 401 | Request bypass gateway langsung ke service hulu tanpa HMAC signature yang sah atau timestamp kedaluwarsa (> 60s). |
| `UPSTREAM_UNAVAILABLE` | 502 | Layanan hilir auth sedang tidak dapat dihubungi. |
| `UPSTREAM_TIMEOUT` | 504 | Layanan hilir melebihi batas waktu eksekusi (timeout). |
| `INTERNAL_ERROR` | 500 | Kesalahan internal server tak terduga. |
| `INVALID_STATE` | 409 | Transisi status pendaftaran ilegal atau di luar alur state machine. |
| `QUOTA_EXCEEDED` | 409 | Kuota pendaftaran atau kuota kelulusan prodi telah terpenuhi. |
| `PERIOD_CLOSED` | 400 | Gelombang pendaftaran belum dibuka atau sudah ditutup. |
| `DUPLICATE_APPLICATION` | 409 | Pengguna sudah memiliki pendaftaran aktif pada gelombang yang sama. |
| `DOCUMENT_REQUIRED` | 422 | Dokumen wajib persyaratan belum diunggah sebelum submit pendaftaran. |
| `BIODATA_INCOMPLETE` | 422 | Profil pendaftar, alamat KTP, ortu, atau pendidikan belum lengkap. |
| `PAYMENT_REQUIRED` | 400 | Tindakan memerlukan pelunasan tagihan terlebih dahulu. |
| `INSTALLMENT_NOT_ALLOWED` | 400 | Skema biaya pendaftaran atau prodi tidak mengizinkan opsi cicilan. |
| `SCORES_LOCKED` | 409 | Nilai seleksi telah dikunci karena rekomendasi prodi telah disubmit. |
| `RESULT_NOT_PUBLISHED` | 403 | Pengumuman hasil seleksi belum dipublikasikan secara resmi. |
| `FILE_TYPE_NOT_ALLOWED` | 422 | Format berkas bukan PDF, JPG, atau PNG yang sah (validasi MIME magic header). |
| `FILE_TOO_LARGE` | 422 | Ukuran berkas melebihi batas maksimal (maks 2 MB). |
| `WEBHOOK_SIGNATURE_INVALID` | 401 | Tanda tangan verifikasi webhook payment gateway tidak sah. |

---

## 3. Rincian Endpoint — Autentikasi Publik & Calon Mahasiswa

### 3.1 Pendaftaran Calon Mahasiswa
- **Method / URL**: `POST /api/v1/auth/register`
- **Autentikasi**: Publik
- **Request Body**:
  ```json
  {
    "email": "calon_mhs@example.com",
    "password": "Password123!",
    "password_confirmation": "Password123!",
    "full_name": "Ahmad Dani",
    "phone": "081234567890"
  }
  ```
- **Respons Sukses (201 Created)**:
  ```json
  {
    "success": true,
    "message": "Pendaftaran berhasil, silakan periksa email Anda untuk kode OTP",
    "data": {
      "email": "calon_mhs@example.com",
      "otp_expires_at": "2026-09-19T08:30:00Z",
      "resend_available_at": "2026-09-19T08:21:00Z"
    }
  }
  ```

### 3.2 Verifikasi OTP (Auto-Login)
- **Method / URL**: `POST /api/v1/auth/verify`
- **Autentikasi**: Publik
- **Request Body**:
  ```json
  {
    "email": "calon_mhs@example.com",
    "otp": "123456"
  }
  ```
- **Respons Sukses (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Verifikasi email berhasil, Anda telah masuk ke sistem",
    "data": {
      "access_token": "eyJhbGciOiJSUzI1NiIs...",
      "refresh_token": "eyJhbGciOiJSUzI1NiIs...",
      "token_type": "Bearer",
      "expires_in": 900,
      "user": {
        "id": "e2c34914-9e32-474c-81be-3c9735d10d65",
        "email": "calon_mhs@example.com",
        "full_name": "Ahmad Dani",
        "phone": "+6281234567890",
        "roles": ["calon_mahasiswa"],
        "scopes": [],
        "status": "active"
      }
    }
  }
  ```

### 3.3 Kirim Ulang Kode OTP
- **Method / URL**: `POST /api/v1/auth/resend-otp`
- **Autentikasi**: Publik (dibatasi cooldown 60 detik)
- **Request Body**:
  ```json
  {
    "email": "calon_mhs@example.com"
  }
  ```
- **Respons Sukses (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Jika email terdaftar, kode OTP baru telah dikirimkan",
    "data": null
  }
  ```

### 3.4 Login Akun
- **Method / URL**: `POST /api/v1/auth/login`
- **Autentikasi**: Publik
- **Request Body**:
  ```json
  {
    "email": "calon_mhs@example.com",
    "password": "Password123!"
  }
  ```
- **Respons Sukses (200 OK)**: Mengembalikan struktur yang sama dengan verifikasi OTP (`access_token`, `refresh_token`, `user`).

### 3.5 Refresh Token (Rotasi Token)
- **Method / URL**: `POST /api/v1/auth/refresh`
- **Autentikasi**: Publik
- **Request Body**:
  ```json
  {
    "refresh_token": "eyJhbGciOiJSUzI1NiIs..."
  }
  ```
- **Respons Sukses (200 OK)**: Mengembalikan pasangan token baru:
  ```json
  {
    "success": true,
    "message": "Token berhasil diperbarui",
    "data": {
      "access_token": "eyJhbGciOiJSUzI1NiIs...NEW...",
      "refresh_token": "eyJhbGciOiJSUzI1NiIs...NEW...",
      "token_type": "Bearer",
      "expires_in": 900,
      "user": { ... }
    }
  }
  ```
> [!IMPORTANT]
> **Deteksi Token Reuse**: Refresh token lama langsung diinvaliasi setelah rotasi. Jika token lama dipakai kembali, server akan menganggap terjadi pencurian sesi, mencabut seluruh sesi user di perangkat tersebut, dan membalas dengan `401 TOKEN_REVOKED`.

### 3.6 Lupa Password
- **Method / URL**: `POST /api/v1/auth/forgot-password`
- **Autentikasi**: Publik
- **Request Body**:
  ```json
  {
    "email": "calon_mhs@example.com"
  }
  ```
- **Respons Sukses (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Jika email terdaftar, instruksi reset kata sandi telah dikirimkan",
    "data": null
  }
  ```

### 3.7 Reset Password
- **Method / URL**: `POST /api/v1/auth/reset-password`
- **Autentikasi**: Publik
- **Request Body**:
  ```json
  {
    "token": "a1b2c3d4e5f6...",
    "password": "NewPassword123!",
    "password_confirmation": "NewPassword123!"
  }
  ```
- **Respons Sukses (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Kata sandi berhasil diperbarui, silakan masuk kembali",
    "data": null
  }
  ```

---

## 4. Rincian Endpoint — Profil Akun Pengguna

### 4.1 Mengambil Profil Akun Pengguna Aktif
- **Method / URL**: `GET /api/v1/auth/me`
- **Header**: `Authorization: Bearer <access_token>`
- **Respons Sukses (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Profil berhasil diambil",
    "data": {
      "id": "e2c34914-9e32-474c-81be-3c9735d10d65",
      "email": "calon_mhs@example.com",
      "full_name": "Ahmad Dani",
      "phone": "+6281234567890",
      "roles": ["calon_mahasiswa"],
      "scopes": [],
      "status": "active"
    }
  }
  ```

### 4.2 Ubah Profil Akun (Display Name & Phone)
- **Method / URL**: `PUT /api/v1/auth/profile`
- **Header**: `Authorization: Bearer <access_token>`
- **Request Body**:
  ```json
  {
    "full_name": "Ahmad Dani Pratama",
    "phone": "081299998888"
  }
  ```

### 4.3 Ganti Password Mandiri
- **Method / URL**: `PUT /api/v1/auth/password`
- **Header**: `Authorization: Bearer <access_token>`
- **Request Body**:
  ```json
  {
    "current_password": "Password123!",
    "password": "PasswordBaru123!",
    "password_confirmation": "PasswordBaru123!"
  }
  ```

### 4.4 Logout Sesi Aktif
- **Method / URL**: `POST /api/v1/auth/logout`
- **Header**: `Authorization: Bearer <access_token>`
- **Request Body (Opsional)**:
  ```json
  {
    "refresh_token": "eyJhbGciOiJSUzI1NiIs..."
  }
  ```

### 4.5 Logout Semua Sesi (Semua Perangkat)
- **Method / URL**: `POST /api/v1/auth/logout-all`
- **Header**: `Authorization: Bearer <access_token>`

---

## 5. Rincian Endpoint — Manajemen Pengguna (Khusus Superadmin)

Prefix: `/api/v1/auth/admin/users`  
Header: `Authorization: Bearer <admin_token>` (Wajib role `superadmin`)

| Method | Endpoint | Deskripsi |
|---|---|---|
| `GET` | `/api/v1/auth/admin/users` | List pengguna dengan search, filter status, filter role, & paginasi |
| `POST` | `/api/v1/auth/admin/users` | Buat user internal baru (admin PMB, keuangan, dosen, prodi, pimpinan) |
| `GET` | `/api/v1/auth/admin/users/:id` | Detail akun pengguna beserta riwayat role dan scope |
| `PUT` | `/api/v1/auth/admin/users/:id` | Perbarui data identitas dasar user |
| `PUT` | `/api/v1/auth/admin/users/:id/roles` | Tetapkan role dan scope baru (contoh prodi: `TI`) |
| `POST` | `/api/v1/auth/admin/users/:id/deactivate` | Nonaktifkan akun pengguna (status `inactive`) |
| `POST` | `/api/v1/auth/admin/users/:id/reactivate` | Aktifkan kembali akun yang dinonaktifkan |

---

## 6. Panduan Implementasi State Token & Interceptor Frontend

Untuk pengalaman pengguna yang mulus (*seamless session*), frontend (Vue/React) disarankan menerapkan HTTP Interceptor dengan alur kerja berikut:

```
[Request HTTP]
       |
       v
[Respons 401?]
  ├── Tidak ──> Lanjutkan proses normal
  └── Ya ─────> Apakah error_code === "TOKEN_EXPIRED"?
                  ├── Tidak ──> Redirect ke Halaman Login (Bersihkan LocalStorage)
                  └── Ya ─────> Panggil POST /api/v1/auth/refresh
                                  ├── Sukses (200):
                                  │     - Simpan access_token & refresh_token baru
                                  │     - Ulangi request asli yang sempat tertunda
                                  └── Gagal (401):
                                        - Redirect ke Login
```

---

## 7. Tabel Migrasi: API Legacy (EduPilot/Jatidiri) vs PMB Baru

| Fitur / Endpoint Lama | Endpoint Baru PMB | Penyesuaian Utama & Alasan Keamanan |
|---|---|---|
| `POST /auth/register` (body `role`) | `POST /auth/register` | Field `role` dihapus dari request klien; pendaftar publik otomatis mendapatkan role `calon_mahasiswa`. |
| `POST /auth/verify` (butuh Bearer token) | `POST /auth/verify` | Bersifat **publik** (`email` + `otp`). Menerapkan batas maksimal 5 percobaan (`OTP_LOCKED`). Sukses langsung mengembalikan token (auto-login). |
| `POST /auth/resend-otp` | `POST /auth/resend-otp` | Publik (`email`), menerapkan anti-spam cooldown 60 detik. |
| `POST /auth/login` | `POST /auth/login` | Pesan error disamakan antara email salah vs password salah (`INVALID_CREDENTIALS`). Dilengkapi proteksi lockout 15 menit jika 5x berturut-turut salah. |
| `POST /auth/refresh` | `POST /auth/refresh` | Menerapkan rotasi token penuh, deteksi penggunaan ulang (*reuse detection*), dan denylist Redis. |
| `POST /auth/logout` | `POST /auth/logout` & `/logout-all` | Token dimasukkan ke denylist Redis sampai masa berlaku habis; mendukung logout multi-perangkat. |
| `GET /auth/me` | `GET /auth/me` | Format seragam, klaim role dan scope terintegrasi, entitas legacy `customer` dihilangkan. |
| `PUT /auth/profile/:id` | **Dihapus (Celah IDOR)** | User hanya boleh mengedit profil diri sendiri (`PUT /auth/profile`). Pengeditan user lain dialihkan ke endpoint admin terproteksi. |
| `PUT /auth/change-role` | **Dihapus (Privilege Escalation)** | Pengguna tidak dapat menaikkan role sendiri. Perubahan role hanya dapat dilakukan superadmin via `/admin/users/:id/roles`. |
| Endpoint `/internal/*` | **Diblokir di Gateway** | Klien publik ditolak HTTP 404 jika mencoba mengakses endpoint internal. Hanya dapat diakses komunikasi antar-service dengan `X-Internal-Token`. |
