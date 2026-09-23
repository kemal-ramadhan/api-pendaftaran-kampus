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

---

## 8. Rincian Endpoint — OCR Service (Internal)

Layanan pengenalan teks KTP otomatis dan peningkatan kualitas citra berbasis *content-preserving* (FastAPI + Python 3.12). Layanan ini **tidak diekspos ke publik oleh API Gateway** dan hanya dapat dipanggil oleh service internal (seperti Identity Service).

### 8.1 Membuat Job OCR KTP
- **Method / URL**: `POST /internal/ocr/ktp`
- **Header**:
  - `X-Internal-Token: <token_internal>`
  - `X-Request-ID: <uuid>`
- **Content-Type**: `multipart/form-data`
- **Body Form-Data**:
  - `file`: Berkas gambar KTP (JPG, PNG, atau PDF halaman 1, maks 5 MB).
- **Respons Sukses (202 Accepted)**:
  ```json
  {
    "success": true,
    "message": "Job OCR berhasil dibuat dan sedang diproses",
    "data": {
      "job_id": "7b01fa2d-7d88-466d-9b09-0ef5ad11b332",
      "status": "processing"
    }
  }
  ```

### 8.2 Mengambil Status dan Hasil Job OCR
- **Method / URL**: `GET /internal/ocr/jobs/:job_id`
- **Header**:
  - `X-Internal-Token: <token_internal>`
  - `X-Request-ID: <uuid>`
- **Respons Sukses — Status `completed` (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Status job OCR berhasil diambil",
    "data": {
      "job_id": "7b01fa2d-7d88-466d-9b09-0ef5ad11b332",
      "status": "completed",
      "quality_score": 88.5,
      "enhanced": false,
      "reason": null,
      "reason_message": null,
      "fields": {
        "nik": { "value": "3273011508050001", "confidence": 0.95, "needs_review": false },
        "nama": { "value": "BUDI SANTOSO", "confidence": 0.90, "needs_review": false },
        "tempat_lahir": { "value": "BANDUNG", "confidence": 0.90, "needs_review": false },
        "tanggal_lahir": { "value": "2005-08-15", "confidence": 0.90, "needs_review": false },
        "jenis_kelamin": { "value": "L", "confidence": 0.90, "needs_review": false },
        "alamat": { "value": "JL. MERDEKA NO. 45", "confidence": 0.90, "needs_review": false },
        "rt_rw": { "value": "003/007", "confidence": 0.90, "needs_review": false },
        "kel_desa": { "value": "BABAKAN CIAMIS", "confidence": 0.90, "needs_review": false },
        "kecamatan": { "value": "SUMUR BANDUNG", "confidence": 0.90, "needs_review": false },
        "agama": { "value": "ISLAM", "confidence": 0.90, "needs_review": false },
        "kewarganegaraan": { "value": "WNI", "confidence": 0.95, "needs_review": false }
      }
    }
  }
  ```
- **Respons Sukses — Status `retake_required` (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Status job OCR berhasil diambil",
    "data": {
      "job_id": "7b01fa2d-7d88-466d-9b09-0ef5ad11b332",
      "status": "retake_required",
      "quality_score": 12.0,
      "enhanced": false,
      "reason": "BLURRY",
      "reason_message": "Foto buram, pegang HP lebih stabil dan pastikan fokus",
      "fields": null
    }
  }
  ```

---

## 6. Rincian Endpoint — Organization Service

Layanan master data kampus untuk mengelola fakultas dan program studi. Digunakan oleh formulir pendaftaran calon mahasiswa, pengaturan scope prodi di Auth, dan downstream services (Admission, NIM, Selection).

### 6.1 Daftar Fakultas (Publik)
- **Method / URL**: `GET /api/v1/organization/faculties`
- **Autentikasi**: Publik (tanpa token)
- **Query Parameter**: `is_active` (opsional, boolean: `true`/`false`)
- **Respons Sukses (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Berhasil mengambil daftar fakultas",
    "data": [
      {
        "id": 1,
        "code": "FT",
        "name": "Fakultas Teknik",
        "is_active": true,
        "created_at": "2026-09-23T10:00:00Z",
        "updated_at": "2026-09-23T10:00:00Z"
      }
    ]
  }
  ```

### 6.2 Daftar Program Studi (Publik)
- **Method / URL**: `GET /api/v1/organization/study-programs`
- **Autentikasi**: Publik
- **Query Parameter**:
  - `faculty_code` (opsional, string misal `FT`)
  - `is_active` (opsional, boolean: `true`/`false`)
- **Respons Sukses (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Berhasil mengambil daftar program studi",
    "data": [
      {
        "id": 1,
        "faculty_id": 1,
        "faculty_code": "FT",
        "faculty_name": "Fakultas Teknik",
        "code": "TI",
        "nim_code": "55",
        "name": "Teknik Informatika",
        "degree": "S1",
        "accreditation": "Unggul",
        "is_active": true,
        "created_at": "2026-09-23T10:00:00Z",
        "updated_at": "2026-09-23T10:00:00Z"
      }
    ]
  }
  ```

### 6.3 Detail Program Studi (Publik)
- **Method / URL**: `GET /api/v1/organization/study-programs/:code`
- **Autentikasi**: Publik
- **Respons Sukses (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Berhasil mengambil detail program studi",
    "data": {
      "id": 1,
      "faculty_id": 1,
      "faculty_code": "FT",
      "faculty_name": "Fakultas Teknik",
      "code": "TI",
      "nim_code": "55",
      "name": "Teknik Informatika",
      "degree": "S1",
      "accreditation": "Unggul",
      "is_active": true
    }
  }
  ```

### 6.4 Tambah Program Studi Baru (Admin)
- **Method / URL**: `POST /api/v1/organization/admin/study-programs`
- **Autentikasi**: Wajib Bearer Token (Role: `superadmin`)
- **Request Body**:
  ```json
  {
    "faculty_id": 1,
    "code": "TE",
    "nim_code": "58",
    "name": "Teknik Elektro",
    "degree": "S1",
    "accreditation": "Unggul",
    "is_active": true
  }
  ```
- **Respons Sukses (201 Created)**:
  ```json
  {
    "success": true,
    "message": "Program studi berhasil dibuat",
    "data": {
      "id": 5,
      "code": "TE",
      "nim_code": "58",
      "name": "Teknik Elektro",
      "degree": "S1",
      "accreditation": "Unggul",
      "is_active": true
    }
  }
  ```

### 6.5 Endpoint Internal (Inter-Service)
Dilindungi header `X-Internal-Token`, tidak diekspos melalui API Gateway:
- `GET /internal/study-programs/:code` $\rightarrow$ Mengambil detail prodi untuk validasi scope role di Auth Service dan kuota pendaftaran di Admission.
- `POST /internal/study-programs/batch` $\rightarrow$ Mengambil detail beberapa prodi sekaligus dengan body `{ "codes": ["TI", "SI"] }`.
