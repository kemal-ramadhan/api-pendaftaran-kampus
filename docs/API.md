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

---

## 7. Identity Service & OCR KTP (F2-M2 & F2-M2b)

Layanan Identity Service mengelola data pribadi (*person identity*), alamat, data orang tua/wali, riwayat pendidikan, dan integrasi pemindaian kartu tanda penduduk (KTP) melalui OCR Service.

### 7.1 Keamanan & Privasi NIK (UU No. 27/2022)
- **At-Rest Encryption**: NIK 16 digit disimpan terenkripsi dengan algoritma **AES-GCM 256-bit** menggunakan kunci aman dari environment `IDENTITY_ENCRYPTION_KEY`.
- **Searchable Index**: Keunikan NIK dijaga melalui kolom `nik_hash` (SHA-256 hexdigest).
- **Masking NIK**: NIK hanya ditampilkan utuh (16 digit) kepada:
  - Pemilik data itu sendiri (`/me/...`)
  - Role `admin_pmb` dan `superadmin` (`/admin/...`)
  - Pemanggilan internal service (`/internal/...`)
  Untuk pihak lain, NIK otomatis dimasking: `3205********0001`.
- **Tanpa Model Generatif**: Peningkatan citra OCR hanya menggunakan algoritma pelestarian isi (*content-preserving*) seperti CLAHE, Unsharp Masking, Bilateral Filter, dan Lanczos Scaling. Tidak ada huruf/angka yang dikarang oleh model AI.
- **Konfirmasi Wajib**: Hasil OCR hanya berupa saran (*suggestions*) dan tidak pernah disimpan otomatis sebelum dikonfirmasi oleh pengguna.

---

### 7.2 Endpoint Calon Mahasiswa (`/api/v1/identity/me/...`)
Seluruh endpoint di bawah memerlukan token JWT calon mahasiswa yang valid (`auth: required`).

#### 1. `GET /api/v1/identity/me/profile`
Mengambil data profil pengguna yang login.
- **Response `200 OK`**:
  ```json
  {
    "success": true,
    "message": "Data profil berhasil diambil",
    "data": {
      "user_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
      "full_name": "Ahmad Syafiq",
      "nik": "3201015505050002",
      "birth_place": "Bandung",
      "birth_date": "2005-05-15",
      "gender": "L",
      "religion": "Islam",
      "nationality": "WNI",
      "phone": "081234567890",
      "photo_url": "https://..."
    }
  }
  ```

#### 2. `PUT /api/v1/identity/me/profile`
Memperbarui data profil pengguna.
- **Request Body**:
  ```json
  {
    "full_name": "Ahmad Syafiq",
    "nik": "3201015505050002",
    "birth_place": "Bandung",
    "birth_date": "2005-05-15",
    "gender": "L",
    "religion": "Islam",
    "nationality": "WNI",
    "phone": "081234567890"
  }
  ```

#### 3. `GET /api/v1/identity/me/addresses` & `PUT /api/v1/identity/me/addresses`
Mengambil dan memperbarui data alamat (tipe `ktp` atau `domisili`).
- **Request Body (PUT)**:
  ```json
  {
    "type": "ktp",
    "street": "Jl. Sukajadi No. 12",
    "rt": "02",
    "rw": "03",
    "village": "Sukagalih",
    "district": "Sukajadi",
    "city": "Bandung",
    "province": "Jawa Barat",
    "postal_code": "40163"
  }
  ```

#### 4. `GET /api/v1/identity/me/guardians` & `PUT /api/v1/identity/me/guardians`
Mengelola data orang tua atau wali (`relation`: `ayah`, `ibu`, `wali`).
- **Request Body (PUT)**:
  ```json
  {
    "relation": "ayah",
    "name": "Bambang Sudarsono",
    "occupation": "Wiraswasta",
    "income_range": "5.000.000 - 10.000.000",
    "phone": "081299887766"
  }
  ```

#### 5. `GET /api/v1/identity/me/education` & `PUT /api/v1/identity/me/education`
Mengelola riwayat pendidikan formal terakhir calon mahasiswa.
- **Request Body (PUT)**:
  ```json
  {
    "level": "SMA",
    "school_name": "SMA Negeri 1 Bandung",
    "npsn": "20219485",
    "major": "IPA",
    "graduation_year": 2024
  }
  ```

#### 6. `PUT /api/v1/identity/me/photo`
Mengunggah pasfoto profil calon mahasiswa (maksimal 2 MB, format JPG/PNG). Disimpan di MinIO bucket `pmb-identity-photos`.
- **Form Data**: `photo` (file gambar).

#### 7. `GET /api/v1/identity/me/completeness`
Memeriksa status kelengkapan biodata.
- **Response `200 OK`**:
  ```json
  {
    "success": true,
    "message": "Status kelengkapan biodata berhasil diperiksa",
    "data": {
      "complete": true,
      "missing": []
    }
  }
  ```
  *(Jika belum lengkap, `complete: false` dan `missing` memuat daftar field yang belum diisi).*

---

### 7.3 Pemindaian & Konfirmasi OCR KTP

#### 1. Inisiasi Scan KTP: `POST /api/v1/identity/me/ktp-scan`
- **Headers**: `Authorization: Bearer <token>`, `Content-Type: multipart/form-data`
- **Form Data**: `file` (gambar scan KTP, maks 5 MB).
- **Response `202 Accepted`**:
  ```json
  {
    "success": true,
    "message": "Proses pembacaan KTP sedang berjalan di latar belakang",
    "data": {
      "scan_id": "7fa6b73a-4422-48df-b452-19e4871987d6",
      "status": "processing"
    }
  }
  ```

#### 2. Polling Status Scan: `GET /api/v1/identity/me/ktp-scan/:scan_id`
- **Response `200 OK` (Selesai)**:
  ```json
  {
    "success": true,
    "message": "Status pemindaian KTP berhasil diambil",
    "data": {
      "scan_id": "7fa6b73a-4422-48df-b452-19e4871987d6",
      "status": "completed",
      "quality_score": 88.5,
      "enhanced": true,
      "fields": {
        "nik": { "value": "3201015505050002", "confidence": 0.98, "needs_review": false },
        "nama": { "value": "AHMAD SYAFIQ", "confidence": 0.95, "needs_review": false },
        "tanggal_lahir": { "value": "2005-05-15", "confidence": 0.99, "needs_review": false }
      }
    }
  }
  ```
- **Response `200 OK` (Foto Terlalu Buram)**:
  ```json
  {
    "success": true,
    "message": "Status pemindaian KTP berhasil diambil",
    "data": {
      "scan_id": "...",
      "status": "retake_required",
      "reason": "BLURRY",
      "reason_message": "Foto buram, pegang HP lebih stabil dan pastikan pencahayaan cukup"
    }
  }
  ```

#### 3. Konfirmasi Hasil OCR: `POST /api/v1/identity/me/ktp-scan/:scan_id/confirm`
Calon mahasiswa memeriksa/mengoreksi data sebelum menyimpannya ke biodata.
- **Request Body**:
  ```json
  {
    "full_name": "Ahmad Syafiq",
    "nik": "3201015505050002",
    "birth_place": "Bandung",
    "birth_date": "2005-05-15",
    "gender": "L",
    "religion": "Islam",
    "nationality": "WNI",
    "phone": "081234567890",
    "address": {
      "type": "ktp",
      "street": "Jl. Sukajadi No. 12",
      "rt": "02",
      "rw": "03",
      "village": "Sukagalih",
      "district": "Sukajadi",
      "city": "Bandung",
      "province": "Jawa Barat",
      "postal_code": "40163"
    }
  }
  ```

---

### 7.4 Endpoint Admin (`/api/v1/identity/admin/...`)
Hanya dapat diakses oleh role `admin_pmb` dan `superadmin`. NIK disajikan secara utuh (*unmasked*).

#### `GET /api/v1/identity/admin/persons/:user_id`
Mengambil detail lengkap identitas calon mahasiswa (profil, daftar alamat, orang tua/wali, riwayat pendidikan, dan status kelengkapan).

---

### 7.5 Endpoint Internal Antar-Service (`/internal/persons/...`)
Dilindungi oleh `X-Internal-Token`:
- `GET /internal/persons/:user_id` $\rightarrow$ Detail identitas lengkap untuk service hilir (Admission, Finance, NIM).
- `GET /internal/persons/:user_id/completeness` $\rightarrow$ Evaluasi kelengkapan berkas untuk validasi sebelum submit pendaftaran di Admission Service.
- `POST /internal/persons/batch` $\rightarrow$ Mengambil sekumpulan identitas pengguna dengan body `{ "user_ids": ["uuid1", "uuid2"] }`.

---

## 8. Rincian Endpoint — Notification Service (Internal Service)

Notification Service (`pmb-notification-service:8083`) melayani pengiriman email dan WhatsApp terpusat. Endpoint dilindungi oleh header `X-Internal-Token`.

### 8.1 Antrean Notifikasi Baru: `POST /internal/notifications`
- **Headers**: `X-Internal-Token: <token>`
- **Request Body**:
  ```json
  {
    "idempotency_key": "app-reg-12345",
    "user_id": "8ca67290-951b-4179-8809-77ea1dfcfae7",
    "channel": "email",
    "email": "pendaftar@example.com",
    "template": "application_registered",
    "data": {
      "name": "Budi Santoso",
      "registration_no": "PMB-2026-1-00001",
      "path_name": "Reguler",
      "period_name": "Gelombang 1 2026/2027",
      "bill_amount": "250000",
      "va_number": "988123456",
      "payment_deadline": "3 hari setelah pendaftaran"
    }
  }
  ```
- **Respons (202 Accepted)**:
  ```json
  {
    "success": true,
    "message": "Notifikasi berhasil dimasukkan ke antrean",
    "data": {
      "id": "c138db53-d144-4866-932f-a2e60cefd48f",
      "status": "queued",
      "channel": "email",
      "template": "application_registered"
    }
  }
  ```

### 8.2 Cek Status Notifikasi: `GET /internal/notifications/:id`
Mengambil status pengiriman, retries, dan pesan kesalahan.

---

## 9. Rincian Endpoint — Admission Service (Pendaftaran Bagian 1 & 2 — F2-M4 & F2-M6)

Admission Service (`pmb-admission-service:8084`) menangani master periode/jalur masuk, kuota, persyaratan berkas, formulir pendaftaran calon mahasiswa, unggah dan validasi dokumen magic bytes ke MinIO, submit pendaftaran terverifikasi biodata, verifikasi berkas per dokumen oleh admin, alur revisi pendaftar, presigned URL dokumen (5 menit untuk admin), ekspor spreadsheet Excel (.xlsx), dan agregasi statistik pendaftaran.

### 9.1 Endpoint Publik / Informasi Pendaftaran

#### `GET /api/v1/admission/periods/active`
Mengambil data gelombang pendaftaran yang sedang dibuka saat ini.
- **Autentikasi**: Publik
- **Respons (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Data periode pendaftaran aktif berhasil diambil",
    "data": {
      "id": 1,
      "name": "Gelombang 1 PMB 2026/2027",
      "academic_year": "2026/2027",
      "start_date": "2026-08-01T00:00:00Z",
      "end_date": "2026-10-31T23:59:59Z",
      "is_active": true
    }
  }
  ```

#### `GET /api/v1/admission/paths`
Mengambil daftar jalur masuk aktif beserta rincian biaya pendaftaran dan dokumen persyaratan.
- **Autentikasi**: Publik

---

### 9.2 Endpoint Pendaftar (`/api/v1/admission/applications/...`)

#### `POST /api/v1/admission/applications`
Membuat formulir pendaftaran baru. Pendaftaran langsung berstatus `awaiting_payment` dan event outbox tagihan pendaftaran dibuat secara transactional.
- **Autentikasi**: Wajib (`calon_mahasiswa`)
- **Request Body**:
  ```json
  {
    "period_id": 1,
    "admission_path_id": 1,
    "choice_1_id": "b1a1c97a-9a91-4cf1-88f5-f5b21ee4bb11",
    "choice_2_id": "d2b3c4e5-1111-4444-8888-abcdef123456"
  }
  ```
- **Error Responses**:
  - `400 PERIOD_CLOSED`: Periode pendaftaran sedang ditutup atau belum dibuka.
  - `409 DUPLICATE_APPLICATION`: Pengguna sudah memiliki pendaftaran pada periode ini.

#### `GET /api/v1/admission/applications/me`
Mengambil riwayat pendaftaran milik akun yang sedang masuk.

#### `GET /api/v1/admission/applications/:id`
Mengambil detail formulir pendaftaran tertentu (hanya milik user bersangkutan).

#### `PUT /api/v1/admission/applications/:id/choices`
Mengubah pilihan program studi 1 & 2.
- **Batasan**: Hanya diizinkan bila status pendaftaran masih `draft` atau `awaiting_payment`.
- **Error `400 INVALID_STATE`**: Muncul bila status sudah `form_open` atau lebih lanjut.

#### `POST /api/v1/admission/applications/me/documents` (atau `/:id/documents`)
Mengunggah berkas dokumen persyaratan ke MinIO privat (`pmb-admission-documents`).
- **Autentikasi**: Wajib (`calon_mahasiswa`)
- **Status yang Diizinkan**: `form_open` atau `revision_required`
- **Content-Type**: `multipart/form-data`
- **Form Data**:
  - `document_type`: Kode jenis dokumen (mis. `ijazah`, `rapor`, `ktp`, `kk`, `pas_foto`, `sertifikat_prestasi`).
  - `file`: Berkas fisik dokumen.
- **Validasi Keamanan Magic Bytes**:
  - Tipe berkas dideteksi secara mendalam dari header byte pertama (`pkg/storage.SniffAndValidate`). Berkas berbahaya (seperti `.exe` atau script biner berkedok `.pdf`) otomatis ditolak dengan `415 Unsupported Media Type` (`FILE_TYPE_NOT_ALLOWED`).
  - Ukuran berkas divalidasi terhadap batas maksimal per dokumen (`FILE_TOO_LARGE`).
- **Respons (201 Created)**:
  ```json
  {
    "success": true,
    "message": "Dokumen persyaratan berhasil diunggah",
    "data": {
      "id": "11111111-2222-3333-4444-555555555555",
      "application_id": "7fa6b73a-4422-48df-b452-19e4871987d6",
      "document_type": "ijazah",
      "file_name": "ijazah_asli.pdf",
      "file_url": "https://...",
      "mime_type": "application/pdf",
      "file_size": 1048576,
      "status": "uploaded"
    }
  }
  ```

#### `DELETE /api/v1/admission/applications/me/documents/:code`
Menghapus dokumen persyaratan yang telah diunggah sebelum submit.
- **Akses**: Calon Mahasiswa (pemilik) saat status `form_open` atau `revision_required`.

#### `GET /api/v1/admission/applications/me/documents/:code/url`
Mengambil presigned download URL sementara (15 menit) untuk berkas pendaftaran milik calon mahasiswa.

#### `POST /api/v1/admission/applications/me/submit` (atau `/:id/submit`)
Mengunci dan mengirimkan seluruh berkas pendaftaran ke panitia PMB.
- **Autentikasi**: Wajib (`calon_mahasiswa`)
- **Status yang Diizinkan**: `form_open` atau `revision_required`
- **Aturan Validasi Ketat**:
  1. **Kelengkapan Biodata**: Memanggil Identity Service (`GET /internal/persons/:user_id/completeness`). Jika belum lengkap, pendaftaran ditolak dengan `400 Bad Request` dan error code `BIODATA_INCOMPLETE` beserta daftar field yang kurang.
  2. **Kelengkapan Berkas Wajib**: Seluruh dokumen dengan `is_required = true` pada jalur pendaftaran terkait wajib telah diunggah dan tidak dalam status `invalid`. Jika ada yang belum diunggah, ditolak dengan `400 Bad Request` dan error code `DOCUMENT_REQUIRED`.
- **Transisi Status**: `form_open` / `revision_required` $\rightarrow$ `submitted`.
- **Notifikasi**: Otomatis mengirim email konfirmasi pendaftaran berhasil disubmit via Notification Service (`application_submitted`).

---

### 9.3 Endpoint Admin & Program Studi (`/api/v1/admission/admin/...`)

#### `GET /api/v1/admission/admin/applications`
Mengambil daftar pendaftar dengan filter `period_id`, `path_id`, `status`, `prodi_id`, `search`, serta pagination.
- **Hak Akses**:
  - `superadmin` / `admin_pmb`: Mengakses seluruh pendaftar.
  - `prodi`: Hanya melihat pendaftar yang prodinya tercantum pada `choice_1_id` atau `choice_2_id`.

#### `GET /api/v1/admission/admin/applications/:id`
Mengambil detail pendaftaran komprehensif untuk panitia PMB.
- **Penggabungan Lintas Service**:
  - Data Formulir & Berkas Dokumen terunggah (Admission Service).
  - Biodata Lengkap calon mahasiswa & NIK utuh/unmasked (Identity Service).
  - Data OCR KTP jika tersedia: URL foto KTP, `quality_score`, serta `fields_edited_by_user`.
  - Status Tagihan & Pembayaran Biaya Pendaftaran (Finance Service).

#### `POST /api/v1/admission/admin/applications/:id/verify`
Memverifikasi dokumen persyaratan pendaftar per berkas.
- **Hak Akses**: `admin_pmb`, `superadmin`
- **Status Pendaftaran yang Diizinkan**: `submitted`
- **Request Body**:
  ```json
  {
    "reviews": [
      {
        "document_type": "ijazah",
        "status": "valid",
        "notes": "Ijazah terlegalisir jelas"
      },
      {
        "document_type": "ktp",
        "status": "invalid",
        "notes": "Foto KTP buram dan nomor NIK tidak terbaca"
      }
    ]
  }
  ```
- **Logika Evaluasi Status**:
  - **Revisi Diperlukan (`revision_required`)**: Jika ada minimal 1 dokumen yang dinyatakan `invalid`. Sistem mencatat catatan revisi dan mengirim notifikasi revisi ke email pendaftar (`application_revision_required`). Calon mahasiswa dapat mengunggah ulang dokumen yang ditolak dan melakukan submit ulang.
  - **Lolos Verifikasi (`verified`)**: Jika seluruh dokumen persyaratan wajib telah berstatus `valid` dan tidak ada dokumen `invalid`. Sistem mengubah status ke `verified` dan mengirim notifikasi kelulusan berkas (`application_verified`).

#### `GET /api/v1/admission/admin/applications/:id/documents/:code/url`
Mengambil presigned URL berkas pendaftar untuk panitia PMB.
- **Keamanan Ketat**: Presigned URL kedaluwarsa tepat **300 detik (5 menit)** untuk meminimalkan risiko kebocoran link dokumen privat.

#### `GET /api/v1/admission/admin/applications/export`
Mengekspor seluruh data pendaftar ke spreadsheet Excel (`.xlsx`).
- **Hak Akses**: `admin_pmb`, `superadmin`
- **Query Filter**: `period_id`, `path_id`, `status`, `prodi_id`, `search`
- **Respons**: File binary stream `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` dengan header `Content-Disposition: attachment; filename="pmb_applications_export_YYYYMMDD_HHMMSS.xlsx"`.
- **Fitur Lembar Kerja**: Styling header biru navy, auto-width kolom, dan format tanggal waktu standar.

#### `GET /api/v1/admission/admin/stats`
Mengambil statistik agregasi pendaftaran PMB secara real-time.
- **Hak Akses**: `admin_pmb`, `superadmin`, `prodi`
- **Respons (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Statistik pendaftaran berhasil diambil",
    "data": {
      "total_applications": 1250,
      "by_status": {
        "awaiting_payment": 200,
        "form_open": 350,
        "submitted": 400,
        "revision_required": 50,
        "verified": 250
      },
      "by_study_program": {
        "b1a1c97a-9a91-4cf1-88f5-f5b21ee4bb11": 720,
        "d2b3c4e5-1111-4444-8888-abcdef123456": 530
      },
      "by_admission_path": {
        "1": 800,
        "2": 300,
        "3": 150
      },
      "by_period": {
        "1": 1250
      }
    }
  }
  ```

---

## 10. Finance Service (`/api/v1/invoices`, `/api/v1/receipts`, `/api/v1/finance/admin`)

Finance Service mengelola skema tarif pendaftaran/daftar ulang, pencatatan invoice/tagihan resmi, integrasi pembayaran Virtual Account via Payment Provider, penerimaan transfer manual dengan verifikasi staf keuangan, penanganan webhook idempoten, dan penerbitan kwitansi elektronik resmi (PDF).

### 10.1 Aturan & Karakteristik Data Keuangan
1. **Nominal Selalu Integer Rupiah**: Semua nilai uang menggunakan bilangan bulat (`int64` / `BIGINT`), tanpa koma/pecahan sen, untuk mencegah floating point rounding error.
2. **Isolasi Tagihan Pengguna**: Pengguna non-admin hanya berhak melihat dan membayar tagihan miliknya sendiri. Akses ke tagihan milik pengguna lain menghasilkan status `403 Forbidden` / `404 Not Found`.
3. **Penyelarasan Status Pendaftaran**: Pelunasan tagihan pendaftaran otomatis mengubah status berkas pendaftaran di Admission Service menjadi `form_open` melalui outbox/internal HTTP callback.

---

### 10.2 Endpoint Tagihan Pengguna (`/api/v1/invoices/me`)

#### `GET /api/v1/invoices/me`
Mengambil daftar seluruh tagihan yang diterbitkan untuk akun yang sedang masuk.
- **Akses**: Calon Mahasiswa (`applicant`)
- **Respons (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Daftar tagihan berhasil diambil",
    "data": [
      {
        "id": "2b3a4c5d-6e7f-8a9b-0c1d-2e3f4a5b6c7d",
        "invoice_number": "INV-20260923-00001",
        "fee_type": "registration",
        "amount": 250000,
        "paid_amount": 0,
        "status": "unpaid",
        "due_date": "2026-09-26T23:59:59Z",
        "description": "Biaya Pendaftaran PMB Gelombang 1"
      }
    ]
  }
  ```

#### `GET /api/v1/invoices/me/:id`
Mengambil detail tagihan tertentu beserta riwayat pembayaran terkait.
- **Akses**: Calon Mahasiswa (pemilik) atau staf `keuangan` / `superadmin`
- **Error Responses**:
  - `403 FORBIDDEN`: Anda tidak memiliki hak akses ke data tagihan ini.
  - `404 NOT_FOUND`: Tagihan tidak ditemukan.

---

### 10.3 Endpoint Pembayaran Virtual Account

#### `POST /api/v1/invoices/me/:id/pay`
Membuat atau mengambil data transaksi pembayaran Virtual Account aktif untuk tagihan tersebut.
- **Akses**: Calon Mahasiswa (pemilik)
- **Request Body**:
  ```json
  {
    "method": "virtual_account"
  }
  ```
- **Respons (200 OK / 201 Created)**:
  ```json
  {
    "success": true,
    "message": "Instruksi pembayaran Virtual Account berhasil dibuat",
    "data": {
      "id": "4c5d6e7f-8a9b-0c1d-2e3f-4a5b6c7d8e9f",
      "invoice_id": "2b3a4c5d-6e7f-8a9b-0c1d-2e3f4a5b6c7d",
      "method": "virtual_account",
      "provider": "mock",
      "va_number": "8808123456789012",
      "amount": 250000,
      "status": "pending",
      "provider_ref": "MOCK-VA-123456"
    }
  }
  ```

---

### 10.4 Endpoint Pembayaran Transfer Manual

#### `POST /api/v1/invoices/me/:id/manual-transfer`
Mengunggah struk/bukti transfer bank manual untuk diverifikasi oleh staf keuangan kampus.
- **Akses**: Calon Mahasiswa (pemilik)
- **Content-Type**: `multipart/form-data`
- **Form Data**:
  - `proof`: File gambar bukti transfer (`image/jpeg`, `image/png`, maks 5 MB).
- **Respons (201 Created)**:
  ```json
  {
    "success": true,
    "message": "Bukti transfer berhasil diunggah, menunggu verifikasi staf keuangan",
    "data": {
      "id": "5d6e7f8a-9b0c-1d2e-3f4a-5b6c7d8e9f0a",
      "method": "manual_transfer",
      "status": "awaiting_verification",
      "amount": 250000,
      "proof_url": "http://localhost:9000/pmb-finance-proofs/proofs/..."
    }
  }
  ```

---

### 10.5 Endpoint Kwitansi Resmi PDF

#### `GET /api/v1/receipts/me/:payment_id`
Mengunduh dokumen kwitansi pembayaran resmi berformat PDF asli.
- **Akses**: Calon Mahasiswa (pemilik pembayaran lunas) atau staf `keuangan`
- **Respons**: Binary stream dengan header:
  - `Content-Type: application/pdf`
  - `Content-Disposition: attachment; filename="Kwitansi-KW-YYYYMMDD-XXXXX.pdf"`

---

### 10.6 Endpoint Simulasi Development (`/api/v1/dev/payments/:id/simulate-paid`)

#### `POST /api/v1/dev/payments/:id/simulate-paid`
Mensimulasikan pelunasan pembayaran secara instan pada lingkungan pengujian/development.
- **Akses**: Khusus lingkungan development (`APP_ENV=development`). Pada lingkungan production akan menghasilkan `403 Forbidden` / `404 Not Found`.
- **Respons (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Simulasi pembayaran berhasil, tagihan lunas dan kwitansi diterbitkan",
    "data": {
      "payment_id": "4c5d6e7f-8a9b-0c1d-2e3f-4a5b6c7d8e9f",
      "status": "success",
      "paid_at": "2026-09-23T19:00:00Z"
    }
  }
  ```

---

### 10.7 Endpoint Webhook Provider (`/api/v1/webhooks/payments/:provider`)

#### `POST /api/v1/webhooks/payments/:provider`
Menerima callback status pembayaran dari payment gateway (misal `mock`, `midtrans`, `xendit`).
- **Akses**: Publik (diverifikasi menggunakan HMAC SHA256 signature pada header `X-Provider-Signature`)
- **Header**:
  - `X-Provider-Signature`: Signature HMAC SHA256 dari payload JSON mentah.
- **Sifat**: **Idempoten**. Payload dengan `provider_ref` yang sama yang dikirim berulang kali hanya akan diproses satu kali tanpa menghasilkan duplikasi pencatatan.

---

### 10.8 Endpoint Staf Keuangan (`/api/v1/finance/admin/...`)

#### `GET /api/v1/finance/admin/payments?status=awaiting_verification`
Mengambil antrean transaksi transfer manual yang menunggu verifikasi staf.
- **Akses**: Staf `keuangan` dan `superadmin`

#### `POST /api/v1/finance/admin/payments/:id/verify`
Menyetujui atau menolak bukti transfer bank.
- **Akses**: Staf `keuangan` dan `superadmin`
- **Request Body**:
  ```json
  {
    "action": "approve", // atau "reject"
    "notes": "Bukti transfer valid dari Bank Mandiri an Budi Prakoso"
  }
  ```
- **Catatan**: Jika ditolak (`reject`), status pembayaran menjadi `rejected` dan calon mahasiswa dapat mengunggah bukti baru. Jika disetujui (`approve`), tagihan lunas dan kwitansi terbit.

---

### 10.9 Endpoint Pengajuan & Jadwal Cicilan Mahasiswa (`/api/v1/invoices/me/...`)

Calon mahasiswa dapat mengajukan permohonan pembayaran tagihan daftar ulang secara bertahap (cicilan/installment). Sesuai PRD, tagihan biaya pendaftaran (`fee_type = "registration"`) ditolak secara mutlak untuk dicicil.

#### `POST /api/v1/invoices/me/:id/installment-request`
Mengajukan permohonan cicilan untuk tagihan tertentu (default maksimal 4 termin, minimal 25% untuk termin pertama).
- **Akses**: Calon Mahasiswa (pemilik tagihan)
- **Request Body**:
  ```json
  {
    "num_terms": 3,
    "reason": "Memohon keringanan pembayaran bertahap 3 bulan"
  }
  ```
- **Validasi & Error**:
  - `422 Unprocessable Entity (INSTALLMENT_NOT_ALLOWED)`: Tagihan biaya pendaftaran (`registration`) tidak dapat dicicil, atau tagihan sudah memiliki status lunas/terkait pengajuan cicilan lain.
  - `422 Unprocessable Entity (VALIDATION_ERROR)`: Jumlah termin harus antara 2 hingga 4 termin.
- **Respons (201 Created)**:
  ```json
  {
    "success": true,
    "message": "Pengajuan cicilan berhasil dikirimkan, menunggu persetujuan bagian keuangan",
    "data": {
      "id": "7f8a9b0c-1d2e-3f4a-5b6c-7d8e9f0a1b2c",
      "invoice_id": "2b3a4c5d-6e7f-8a9b-0c1d-2e3f4a5b6c7d",
      "user_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
      "num_terms": 3,
      "reason": "Memohon keringanan pembayaran bertahap 3 bulan",
      "status": "pending",
      "created_at": "2026-09-24T06:00:00Z"
    }
  }
  ```

#### `GET /api/v1/invoices/me/:id/installments`
Mengambil jadwal termin cicilan aktif untuk tagihan yang pengajuan cicilannya telah disetujui.
- **Akses**: Calon Mahasiswa (pemilik) atau staf `keuangan` / `superadmin`
- **Respons (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Jadwal cicilan berhasil diambil",
    "data": [
      {
        "id": "11111111-2222-3333-4444-555555555551",
        "invoice_id": "2b3a4c5d-6e7f-8a9b-0c1d-2e3f4a5b6c7d",
        "term_number": 1,
        "amount": 1111112,
        "due_date": "2026-10-24T06:00:00Z",
        "status": "unpaid",
        "paid_at": null
      },
      {
        "id": "11111111-2222-3333-4444-555555555552",
        "invoice_id": "2b3a4c5d-6e7f-8a9b-0c1d-2e3f4a5b6c7d",
        "term_number": 2,
        "amount": 1111111,
        "due_date": "2026-11-23T06:00:00Z",
        "status": "unpaid",
        "paid_at": null
      },
      {
        "id": "11111111-2222-3333-4444-555555555553",
        "invoice_id": "2b3a4c5d-6e7f-8a9b-0c1d-2e3f4a5b6c7d",
        "term_number": 3,
        "amount": 1111111,
        "due_date": "2026-12-23T06:00:00Z",
        "status": "unpaid",
        "paid_at": null
      }
    ]
  }
  ```
- **Catatan Pembayaran Per Termin**:
  - Saat membayar via `POST /api/v1/invoices/me/:id/pay`, sistem secara otomatis mengaitkan pembayaran ke termin tertua yang belum dibayar (`installment_id`).
  - Setelah termin 1 dibayar lunas, status invoice berubah menjadi `partially_paid` (memenuhi syarat kelulusan & pendaftaran ulang di Student Service).
  - Setelah seluruh termin dibayar, status invoice berubah menjadi `paid`.

---

### 10.10 Endpoint Peninjauan Cicilan Admin Keuangan (`/api/v1/finance/admin/...`)

#### `GET /api/v1/finance/admin/installment-requests`
Mengambil daftar pengajuan cicilan mahasiswa dengan filter status.
- **Akses**: `keuangan`, `superadmin`, `admin_pmb`
- **Query Params**:
  - `status`: `pending`, `approved`, `rejected` (opsional)
- **Respons (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Daftar pengajuan cicilan berhasil diambil",
    "data": [
      {
        "id": "7f8a9b0c-1d2e-3f4a-5b6c-7d8e9f0a1b2c",
        "invoice_id": "2b3a4c5d-6e7f-8a9b-0c1d-2e3f4a5b6c7d",
        "user_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
        "num_terms": 3,
        "reason": "Memohon keringanan pembayaran bertahap 3 bulan",
        "status": "pending",
        "invoice": {
          "invoice_number": "INV-20260924-00001",
          "amount": 3333334,
          "fee_type": "reregistration"
        }
      }
    ]
  }
  ```

#### `POST /api/v1/finance/admin/installment-requests/:id/review`
Menyetujui atau menolak pengajuan cicilan calon mahasiswa.
- **Akses**: `keuangan`, `superadmin`, `admin_pmb`
- **Request Body**:
  ```json
  {
    "action": "approve", // atau "reject"
    "notes": "Pengajuan cicilan disetujui sesuai regulasi kampus"
  }
  ```
- **Efek Transaksional Saat Approved**:
  1. Status pengajuan menjadi `approved`.
  2. Status skema pembayaran tagihan berubah menjadi `installment`.
  3. Jadwal termin di tabel `installments` digenerate secara transaksional dengan rumus presisi integer rupiah:
     - Termin 1: jatuh tempo 30 hari dari persetujuan (+ sisa pembagian bulat jika ada).
     - Termin $i$: jatuh tempo $30 \times i$ hari dari persetujuan.
     - Total seluruh termin dijamin tepat 100% tanpa selisih Rp 1 pun dari nominal tagihan asli.
  4. Notifikasi persetujuan cicilan dikirimkan ke calon mahasiswa via Notification Service.

---

### 10.11 Endpoint Laporan Keuangan & Ekspor Spreadsheet (`/api/v1/finance/admin/reports/...`)

Disediakan untuk pimpinan universitas dan staf keuangan guna memantau arus kas penerimaan, piutang, dan status pembayaran pendaftar.

#### `GET /api/v1/finance/admin/reports/summary`
Mengambil ringkasan agregat keuangan real-time.
- **Akses**: `keuangan`, `pimpinan`, `superadmin`, `admin_pmb`
- **Query Params**:
  - `start_date`: ISO format tanggal awal (opsional)
  - `end_date`: ISO format tanggal akhir (opsional)
  - `fee_type`: `registration` atau `reregistration` (opsional)
- **Respons (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Ringkasan laporan keuangan berhasil diambil",
    "data": {
      "total_invoices": 150,
      "total_billed": 450000000,
      "total_collected": 375000000,
      "total_outstanding": 75000000,
      "paid_count": 120,
      "unpaid_count": 15,
      "partially_paid_count": 10,
      "expired_count": 5,
      "by_fee_type": {
        "registration": {
          "total_invoices": 100,
          "billed": 25000000,
          "collected": 25000000
        },
        "reregistration": {
          "total_invoices": 50,
          "billed": 425000000,
          "collected": 350000000
        }
      }
    }
  }
  ```

#### `GET /api/v1/finance/admin/reports/export?format=xlsx`
Mengunduh laporan komprehensif keuangan dalam format berkas spreadsheet Excel `.xlsx`.
- **Akses**: `keuangan`, `pimpinan`, `superadmin`, `admin_pmb`
- **Respons**: Binary stream berkas Excel:
  - `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
  - `Content-Disposition: attachment; filename="Laporan_Keuangan_PMB_YYYYMMDD_HHMMSS.xlsx"`
- **Struktur Berkas Excel**:
  - **Sheet 1 ("Ringkasan Keuangan")**: Judul resmi institusi, KPI agregat (Total Ditagihkan, Total Diterima, Total Piutang), dan tabel rekapitulasi per jenis biaya (`registration` vs `reregistration`).
  - **Sheet 2 ("Rincian Tagihan")**: Rincian baris per tagihan pendaftar (Nomor Tagihan, User ID, Jenis Biaya, Skema, Jumlah Tagihan, Jumlah Terbayar, Status, Tanggal Terbit, Jatuh Tempo, dan Keterangan).

---

### 10.12 Pekerjaan Terjadwal & Pengingat Jatuh Tempo (`/api/v1/finance/admin/cron/...`)

#### `POST /api/v1/finance/admin/cron/process-due-dates`
Memproses pemeriksaan jatuh tempo secara periodik (cron job):
1. **Pengingat H-3**: Mengirimkan notifikasi email pengingat tagihan/termin yang akan jatuh tempo dalam kurun 3 hari.
2. **Pengingat H-1**: Mengirimkan notifikasi email urgensi tagihan/termin yang akan jatuh tempo besok.
3. **Penandaan Expired**: Mengubah status tagihan/termin yang telah melewati `due_date` dan belum terbayar menjadi `expired`.
- **Akses**: `keuangan`, `superadmin`, `admin_pmb` (atau trigger cron daemon sistem)
- **Respons (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Pemeriksaan jatuh tempo dan pengiriman pengingat berhasil diproses",
    "data": {
      "reminders_sent": 12,
      "expired_updated": 3
    }
  }
  ```

---

## 11. Exam Service (`pmb-exam-service`)

Exam Service mengelola sesi ujian tulis dan wawancara, penugasan dosen penguji, penjadwalan peserta pendaftar yang telah terverifikasi, pencatatan kehadiran presensi, dan penerbitan kartu tanda peserta ujian resmi (PDF) dengan QR code dinamis.

### 11.1 Manajemen Sesi Ujian (Admin PMB)

#### `POST /api/v1/exam/admin/sessions`
Membuat sesi ujian baru.
- **Akses**: `admin_pmb`, `superadmin`
- **Request Body**:
  ```json
  {
    "period_id": "b0000000-0000-0000-0000-000000000001",
    "path_id": "c0000000-0000-0000-0000-000000000001",
    "type": "written_test", // atau "interview"
    "study_program_code": "TI", // opsional
    "title": "Ujian Tulis Kemampuan Dasar Sesi 1",
    "start_at": "2026-11-01T08:00:00Z",
    "end_at": "2026-11-01T10:00:00Z",
    "location": "Gedung Rektorat Lab Komputer 1",
    "meeting_link": null,
    "capacity": 30
  }
  ```
- **Respons (201 Created)**: Objek sesi ujian yang berhasil dibuat.

#### `GET /api/v1/exam/admin/sessions`
Mendapatkan daftar seluruh sesi ujian dengan filter fleksibel.
- **Akses**: `admin_pmb`, `superadmin`
- **Query Params**: `period_id`, `path_id`, `type`, `is_active`

#### `GET /api/v1/exam/admin/sessions/:id`
Mendapatkan rincian sesi ujian beserta daftar dosen penguji bertugas dan jumlah peserta terdaftar.
- **Akses**: `admin_pmb`, `superadmin`

#### `PUT /api/v1/exam/admin/sessions/:id`
Memperbarui informasi jadwal, kuota, atau lokasi sesi ujian.
- **Akses**: `admin_pmb`, `superadmin`

#### `DELETE /api/v1/exam/admin/sessions/:id`
Menghapus sesi ujian (hanya diizinkan jika belum memiliki peserta terdaftar).
- **Akses**: `admin_pmb`, `superadmin`

---

### 11.2 Penugasan Dosen Penguji

#### `POST /api/v1/exam/admin/sessions/:id/examiners`
Menugaskan dosen penguji ke sesi ujian tertentu.
- **Akses**: `admin_pmb`, `superadmin`
- **Request Body**:
  ```json
  {
    "examiner_user_id": "d0000000-0000-0000-0000-000000000001"
  }
  ```

#### `DELETE /api/v1/exam/admin/sessions/:id/examiners/:user_id`
Mencabut penugasan dosen dari sesi ujian.
- **Akses**: `admin_pmb`, `superadmin`

---

### 11.3 Penjadwalan Peserta Ujian

#### `POST /api/v1/exam/admin/sessions/:id/participants`
Menjadwalkan pendaftar ke sesi ujian secara manual.
- **Akses**: `admin_pmb`, `superadmin`
- **Request Body**:
  ```json
  {
    "application_id": "e0000000-0000-0000-0000-000000000001"
  }
  ```
- **Aturan Guard**:
  - Pendaftar wajib berstatus `verified` di Admission Service. Jika tidak, mengembalikan `422 Unprocessable Entity (APPLICATION_NOT_VERIFIED)`.
  - Jumlah peserta tidak boleh melampaui kapasitas sesi. Jika penuh, mengembalikan `422 Unprocessable Entity (SESSION_CAPACITY_EXCEEDED)`.
  - Satu pendaftar tidak dapat dijadwalkan ganda pada sesi yang sama (`409 Conflict (DUPLICATE_PARTICIPANT)`).
- **Side Effects**:
  - Menerbitkan nomor peserta unik berformat `EXAM-{YYYY}-{urut:05d}`.
  - Menerbitkan kartu tanda peserta ujian (PDF) dengan QR code dan tersimpan di MinIO bucket `pmb-exam-cards`.
  - Otomatis mengubah status pendaftaran di Admission Service menjadi `exam_scheduled`.
  - Mengirimkan email pemberitahuan jadwal & kartu ujian ke Notification Service.

#### `POST /api/v1/exam/admin/sessions/:id/auto-assign`
Menjadwalkan beberapa pendaftar sekaligus (*batch*) ke sesi ujian hingga batas kapasitas sesi terpenuhi.
- **Akses**: `admin_pmb`, `superadmin`
- **Request Body**:
  ```json
  {
    "application_ids": [
      "e0000000-0000-0000-0000-000000000001",
      "e0000000-0000-0000-0000-000000000002"
    ]
  }
  ```

#### `GET /api/v1/exam/admin/sessions/:id/participants`
Mengambil daftar seluruh peserta terdaftar di sesi ujian beserta status presensi.
- **Akses**: `admin_pmb`, `superadmin`

---

### 11.4 Presensi Kehadiran Peserta

#### `PATCH /api/v1/exam/admin/participants/:id/attendance`
Mencatat status kehadiran peserta pada sesi ujian.
- **Akses**: `admin_pmb`, `superadmin`, atau `dosen` yang ditugaskan pada sesi terkait
- **Request Body**:
  ```json
  {
    "attendance": "present" // atau "absent"
  }
  ```
- **Isolasi Keamanan**: Akun dosen hanya dapat mengubah presensi peserta pada sesi yang ditugaskan kepadanya. Jika mengakses sesi dosen lain, mengembalikan `403 Forbidden`.

---

### 11.5 Endpoint Calon Mahasiswa (`/api/v1/exam/me/...`)

#### `GET /api/v1/exam/me/schedule`
Melihat rincian jadwal sesi ujian peserta yang terdaftar untuk akun calon mahasiswa login.
- **Akses**: `calon_mahasiswa`

#### `GET /api/v1/exam/me/card`
Mengunduh berkas Kartu Tanda Peserta Ujian Seleksi Masuk resmi berformat PDF.
- **Akses**: `calon_mahasiswa`
- **Respons**: Binary stream PDF dengan header:
  - `Content-Type: application/pdf`
  - `Content-Disposition: attachment; filename="kartu-ujian-EXAM-YYYY-XXXXX.pdf"`
- **Karakteristik Dokumen**: Memuat nomor peserta resmi, identitas, foto, jadwal, lokasi, tata tertib, dan kode QR 2D yang dapat dipindai oleh pengawas ujian di lokasi.

---

### 11.6 Endpoint Dosen Penguji (`/api/v1/exam/examiner/...`)

#### `GET /api/v1/exam/examiner/sessions`
Mengambil daftar sesi ujian yang ditugaskan khusus untuk dosen penguji yang sedang login.
- **Akses**: `dosen`
- **Isolasi**: Sesi milik dosen lain tidak terlihat.

#### `GET /api/v1/exam/examiner/sessions/:id/participants`
Mengambil daftar peserta pada sesi ujian dosen bersangkutan untuk keperluan verifikasi identitas dan penilaian wawancara/tulis.
- **Akses**: `dosen` (hanya sesi yang ditugaskan)

---

### 11.7 Endpoint Internal Inter-Service (`/internal/...`)

#### `GET /internal/participants/:application_id`
Mengambil data peserta ujian berdasarkan ID pendaftaran (digunakan oleh Scoring Service untuk memverifikasi penugasan dosen penguji).
- **Akses**: Khusus internal (Header `X-Internal-Token`)

#### `GET /internal/sessions/:id`
Mengambil rincian sesi ujian.
- **Akses**: Khusus internal (Header `X-Internal-Token`)

---

## 12. Rincian Endpoint — Scoring Service

Scoring Service (`pmb-scoring-service:8089`) bertanggung jawab mengelola konfigurasi komponen penilaian seleksi per jalur, pencatatan skor peserta oleh penguji terverifikasi, kalkulasi nilai akhir tertimbang (*weighted average*), pemeringkatan dinamis pendaftar per program studi pilihan, penyajian rekapitulasi nilai berbasis scope program studi, ekspor berkas spreadsheet `.xlsx` resmi, serta antarmuka penguncian (*locking*) untuk proses rekomendasi kelulusan.

### 12.1 Konsep Komponen Nilai & Validasi Bobot Mutlak 100%
- Setiap jalur seleksi (`path_id`) memiliki sejumlah komponen penilaian (misal: Ujian Tulis Kemampuan Dasar bobot 60%, Wawancara Motivasi bobot 40%).
- **Validasi Mutlak**: Total persentase bobot (`weight_percent`) seluruh komponen aktif pada satu jalur **wajib tepat 100%**.
  - Penambahan atau pengubahan komponen yang menyebabkan akumulasi bobot melebihi 100% langsung ditolak dengan kode status `422 Unprocessable Entity (INVALID_WEIGHT_SUM)`.
  - Verifikasi total bobot dapat dilakukan melalui endpoint khusus `/api/v1/scoring/admin/paths/:path_id/validate-weight`.

### 12.2 Formula Nilai Akhir & Pemeringkatan Dinamis
1. **Rata-Rata Tertimbang (*Weighted Average*)**:
   $$\text{FinalScore} = \sum_{i=1}^n \left( \text{score}_i \times \frac{\text{weight}_i}{100} \right)$$
   Nilai akhir dibulatkan dengan presisi 2 angka di belakang koma (*two decimal places*).
2. **Pemeringkatan Otomatis (`rank_in_program`)**:
   Setiap kali nilai diinput atau diperbarui, sistem secara transaksional menghitung ulang urutan peringkat di antara pendaftar yang memilih Program Studi Pilihan 1 yang sama pada gelombang/periode yang sama menggunakan *window function* `ROW_NUMBER() OVER (PARTITION BY period_id, path_id, study_program_code ORDER BY final_score DESC, computed_at ASC)`.
3. **Proteksi Penguncian Nilai (`SCORES_LOCKED`)**:
   Apabila kombinasi gelombang dan program studi telah dikunci oleh panitia seleksi (`is_locked = true`), seluruh permintaan perubahan atau penambahan nilai ditolak dengan HTTP `422 Unprocessable Entity` dan kode error `SCORES_LOCKED`.

---

### 12.3 Endpoint Admin PMB (`/api/v1/scoring/admin/...`)
Seluruh endpoint di bawah memerlukan peran `admin_pmb` atau `superadmin`.

#### 1. Tambah Komponen Nilai: `POST /api/v1/scoring/admin/components`
- **Request Body**:
  ```json
  {
    "path_id": "c0000000-0000-0000-0000-000000000001",
    "code": "written_test",
    "name": "Ujian Tulis Berbasis Komputer",
    "weight_percent": 60.00,
    "max_score": 100.00,
    "is_active": true
  }
  ```
- **Response `201 Created`**: Komponen berhasil dibuat.
- **Error `422 Unprocessable Entity`**: Jika akumulasi bobot > 100% (`INVALID_WEIGHT_SUM`).
- **Error `400 Bad Request`**: Jika kode komponen sudah terdaftar pada jalur tersebut (`DUPLICATE_COMPONENT`).

#### 2. Ambil Komponen per Jalur: `GET /api/v1/scoring/admin/components?path_id=:path_id`
- **Response `200 OK`**: Daftar komponen penilaian pada jalur yang dipilih.

#### 3. Perbarui Komponen Nilai: `PUT /api/v1/scoring/admin/components/:id`
- **Request Body**:
  ```json
  {
    "name": "Ujian Tulis & Kemampuan Skolastik",
    "weight_percent": 60.00,
    "max_score": 100.00,
    "is_active": true
  }
  ```

#### 4. Hapus Komponen Nilai: `DELETE /api/v1/scoring/admin/components/:id`
- **Response `200 OK`**: Komponen berhasil dihapus.

#### 5. Validasi Total Bobot Jalur: `POST /api/v1/scoring/admin/paths/:path_id/validate-weight`
- **Response `200 OK`**: Jika total persentase bobot komponen aktif tepat 100.00%.
- **Response `422 Unprocessable Entity`**: Jika total persentase bobot != 100.00%.

---

### 12.4 Endpoint Dosen Penguji (`/api/v1/scoring/examiner/...`)
Dapat diakses oleh role `dosen`, `admin_pmb`, atau `superadmin`.

#### 1. Input / Perbarui Nilai Peserta: `PUT /api/v1/scoring/examiner/scores/:application_id`
- **Request Body**:
  ```json
  {
    "component_code": "written_test",
    "score": 85.50,
    "note": "Menyelesaikan seluruh soal numerik dengan sangat baik"
  }
  ```
- **Aturan Otorisasi & Validasi**:
  - Dosen hanya diizinkan menginput nilai bagi peserta yang terdaftar pada sesi ujian yang ditugaskan kepadanya. Sistem memverifikasi penugasan penguji ke Exam Service secara internal (`GET /internal/participants/:application_id` dan `GET /internal/sessions/:session_id`). Upaya penilaian peserta di luar sesi dosen ditolak dengan HTTP `403 Forbidden`.
  - Jika kombinasi periode dan prodi peserta sudah dikunci, ditolak dengan HTTP `422 Unprocessable Entity (SCORES_LOCKED)`.
  - Nilai tidak boleh bernilai negatif atau melebihi batas `max_score` komponen.
- **Side Effects**:
  - Menyimpan/memperbarui record pada tabel `scores`.
  - Menghitung ulang rata-rata tertimbang (`final_score`) pada `score_summaries`.
  - Menghitung ulang peringkat (`rank_in_program`) seluruh peserta di program studi bersangkutan.
- **Response `200 OK`**:
  ```json
  {
    "success": true,
    "message": "Nilai pendaftar berhasil disimpan dan nilai akhir dihitung",
    "data": {
      "score": {
        "application_id": "e0000000-0000-0000-0000-000000000001",
        "component_code": "written_test",
        "score": 85.50,
        "note": "Menyelesaikan seluruh soal numerik dengan sangat baik"
      },
      "summary": {
        "application_id": "e0000000-0000-0000-0000-000000000001",
        "study_program_code": "TIF",
        "final_score": 85.50,
        "rank_in_program": 1
      }
    }
  }
  ```

#### 2. Tinjau Nilai Peserta: `GET /api/v1/scoring/examiner/scores/:application_id`
- **Response `200 OK`**: Daftar skor per komponen dan ringkasan nilai akhir peserta.

---

### 12.5 Endpoint Rekapitulasi & Ekspor Program Studi (`/api/v1/scoring/program/...`)
Dapat diakses oleh role `prodi`, `admin_pmb`, `superadmin`, dan `pimpinan`.

#### 1. Ambil Rekapitulasi & Ranking: `GET /api/v1/scoring/program/recap`
- **Query Parameters**:
  - `period_id` (UUID, wajib)
  - `path_id` (UUID, wajib)
  - `study_program_code` (String, wajib)
- **Isolasi Hak Akses (Role Prodi Scope Check)**:
  - Pengguna dengan role `prodi` hanya dapat melihat data program studi yang tertera pada `scope_id` tokennya. Upaya mengakses prodi lain ditolak dengan HTTP `403 Forbidden`.
  - Pengguna dengan role `admin_pmb`, `superadmin`, atau `pimpinan` memiliki akses tinjauan global ke seluruh prodi.
- **Response `200 OK`**:
  ```json
  {
    "success": true,
    "message": "Rekapitulasi nilai dan pemeringkatan program studi berhasil diambil",
    "data": {
      "period_id": "b0000000-0000-0000-0000-000000000001",
      "path_id": "c0000000-0000-0000-0000-000000000001",
      "study_program_code": "TIF",
      "is_locked": false,
      "total_applicants": 2,
      "participants": [
        {
          "application_id": "e0000000-0000-0000-0000-000000000001",
          "registration_number": "REG-2026-001",
          "full_name": "Ahmad Syafiq",
          "study_program_code": "TIF",
          "component_scores": [
            { "component_code": "written_test", "component_name": "Ujian Tulis", "weight_percent": 60, "score": 85.00 },
            { "component_code": "interview", "component_name": "Wawancara", "weight_percent": 40, "score": 90.00 }
          ],
          "final_score": 87.00,
          "rank_in_program": 1
        }
      ]
    }
  }
  ```

#### 2. Ekspor Rekapitulasi ke Excel (.xlsx): `GET /api/v1/scoring/program/recap/export`
- **Query Parameters**: `period_id`, `path_id`, `study_program_code`.
- **Response `200 OK`**: Berkas spreadsheet `.xlsx` resmi yang digenerate langsung menggunakan library `excelize/v2` dengan header styling navy blue, tabel border, format angka desimal rapi, rincian komponen nilai, nilai akhir, serta kolom peringkat.
  - Header: `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
  - Header: `Content-Disposition: attachment; filename="rekap-nilai-<study_program_code>.xlsx"`

---

### 12.6 Endpoint Internal Antarlayanan (`/internal/...`)
Khusus dipanggil oleh Selection Service dengan menyertakan header `X-Internal-Token`.

#### 1. Ambil Ringkasan Nilai: `GET /internal/summaries?period_id=:period_id&program=:study_program_code`
- Digunakan oleh Selection Service saat merumuskan rekomendasi kelulusan berbasis kuota prodi.

#### 2. Atur Status Penguncian Nilai: `POST /internal/locks`
- **Request Body**:
  ```json
  {
    "period_id": "b0000000-0000-0000-0000-000000000001",
    "path_id": "c0000000-0000-0000-0000-000000000001",
    "study_program_code": "TIF",
    "is_locked": true,
    "locked_by": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"
  }
  ```
- **Response `200 OK`**: Mengunci atau membuka kembali izin penginputan nilai pendaftar pada gelombang dan prodi terkait.

#### 3. Periksa Status Kunci: `GET /internal/locks?period_id=&path_id=&study_program_code=`
- **Response `200 OK`**: Mengembalikan objek boolean `is_locked`.

---

## 13. Selection Service API

Selection Service mengelola alur penetapan kelulusan calon mahasiswa, dimulai dari perumusan rekomendasi oleh program studi berbasis kuota mutlak, penguncian nilai transaksional, persetujuan pimpinan (approve/return), hingga publikasi pengumuman kelulusan resmi oleh Admin PMB dengan side-effects terintegrasi (Admission status update, permintaan invoice daftar ulang ke Finance Service, dan pengiriman notifikasi pengumuman).

Base Path: `/api/v1/selection`

### 13.1 Endpoint Rekomendasi Program Studi (`/api/v1/selection/program`)
Akses: Role `prodi` (sesuai scope prodi masing-masing), `admin_pmb`, `superadmin`.

#### 1. Ambil Rekomendasi & Rekap Peserta: `GET /api/v1/selection/program/recommendations`
Mengambil data usulan kelulusan beserta rekap pendaftar terurut berdasarkan peringkat dan nilai akhir dari Scoring Service.
- **Query Parameters**:
  - `period_id` (UUID, wajib)
  - `path_id` (UUID, wajib)
  - `study_program_code` (string, wajib)
- **Response `200 OK`**:
  ```json
  {
    "success": true,
    "message": "Data rekomendasi program studi berhasil diambil",
    "data": {
      "recommendation": {
        "id": "e0000000-0000-0000-0000-000000000001",
        "period_id": "b0000000-0000-0000-0000-000000000001",
        "path_id": "c0000000-0000-0000-0000-000000000001",
        "study_program_code": "TIF",
        "status": "draft",
        "items": [
          {
            "id": "f0000000-0000-0000-0000-000000000001",
            "application_id": "a0000000-0000-0000-0000-000000000001",
            "decision": "pass",
            "final_score": 87.50,
            "rank": 1
          }
        ]
      },
      "participants": [
        {
          "application_id": "a0000000-0000-0000-0000-000000000001",
          "registration_number": "PMB-2026-0001",
          "full_name": "Budi Santoso",
          "study_program_code": "TIF",
          "final_score": 87.50,
          "rank_in_program": 1,
          "decision": "pass"
        }
      ]
    }
  }
  ```

#### 2. Simpan Usulan Rekomendasi: `POST /api/v1/selection/program/recommendations` (atau `PUT`)
Menyimpan rancangan usulan kelulusan peserta. Kuota program studi divalidasi secara mutlak dari Admission Service.
- **Request Body**:
  ```json
  {
    "period_id": "b0000000-0000-0000-0000-000000000001",
    "path_id": "c0000000-0000-0000-0000-000000000001",
    "study_program_code": "TIF",
    "items": [
      {
        "application_id": "a0000000-0000-0000-0000-000000000001",
        "decision": "pass",
        "final_score": 87.50,
        "rank": 1
      },
      {
        "application_id": "a0000000-0000-0000-0000-000000000002",
        "decision": "fail",
        "final_score": 62.00,
        "rank": 2
      }
    ]
  }
  ```
- **Error `422 Unprocessable Entity` (Jika Lulus Melebihi Kuota)**:
  ```json
  {
    "success": false,
    "message": "QUOTA_EXCEEDED: jumlah peserta yang direkomendasikan lulus melebihi kuota program studi",
    "error_code": "QUOTA_EXCEEDED"
  }
  ```

#### 3. Ajukan Rekomendasi ke Pimpinan: `POST /api/v1/selection/program/recommendations/:id/submit`
Mengubah status usulan menjadi `submitted` dan memanggil Scoring Service (`POST /internal/locks`) untuk mengunci seluruh nilai peserta pada prodi tersebut (`SCORES_LOCKED`).
- **Response `200 OK`**:
  ```json
  {
    "success": true,
    "message": "Rekomendasi berhasil diajukan ke pimpinan dan nilai peserta telah dikunci",
    "data": {
      "id": "e0000000-0000-0000-0000-000000000001",
      "status": "submitted"
    }
  }
  ```

---

### 13.2 Endpoint Persetujuan Pimpinan (`/api/v1/selection/approval`)
Akses: Role `pimpinan`, `superadmin`.

#### 1. Daftar Rekomendasi Menunggu Persetujuan: `GET /api/v1/selection/approval/recommendations`
- **Query Parameters**: `period_id` (opsional)
- **Response `200 OK`**: Daftar usulan berstatus `submitted` beserta item pendaftar.

#### 2. Setujui Rekomendasi: `POST /api/v1/selection/approval/recommendations/:id/approve`
Pimpinan menyetujui rekomendasi kelulusan dari prodi. Status berubah menjadi `approved`.
- **Response `200 OK`**:
  ```json
  {
    "success": true,
    "message": "Rekomendasi program studi berhasil disetujui",
    "data": {
      "id": "e0000000-0000-0000-0000-000000000001",
      "status": "approved",
      "approved_by": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"
    }
  }
  ```

#### 3. Kembalikan Rekomendasi untuk Revisi: `POST /api/v1/selection/approval/recommendations/:id/return`
Pimpinan menolak/mengembalikan rekomendasi dengan catatan revisi. Status berubah menjadi `returned` dan kunci nilai di Scoring Service dibuka kembali (`is_locked = false`) agar prodi/dosen dapat melakukan penyesuaian nilai atau keputusan.
- **Request Body**:
  ```json
  {
    "note": "Mohon pertimbangkan kembali pendaftar cadangan peringkat 3"
  }
  ```
- **Response `200 OK`**:
  ```json
  {
    "success": true,
    "message": "Rekomendasi dikembalikan ke program studi untuk revisi dan nilai telah dibuka kembali",
    "data": {
      "id": "e0000000-0000-0000-0000-000000000001",
      "status": "returned",
      "note": "Mohon pertimbangkan kembali pendaftar cadangan peringkat 3"
    }
  }
  ```

---

### 13.3 Endpoint Publikasi Admin PMB (`/api/v1/selection/admin`)
Akses: Role `admin_pmb`, `superadmin`.

#### 1. Publikasi Pengumuman Kelulusan: `POST /api/v1/selection/admin/announcements`
Memvalidasi bahwa seluruh usulan pada periode tersebut telah `approved`. Melakukan batch insert penetapan ke tabel `results`, memperbarui status pendaftar di Admission Service (`passed` / `failed`), memicu pembuatan tagihan daftar ulang di Finance Service (`reregistration`), serta mengirim notifikasi pengumuman seleksi.
- **Request Body**:
  ```json
  {
    "period_id": "b0000000-0000-0000-0000-000000000001",
    "title": "Pengumuman Hasil Seleksi PMB Gelombang 1 TA 2026/2027",
    "content": "Selamat kepada calon mahasiswa yang dinyatakan LULUS seleksi..."
  }
  ```
- **Response `201 Created`**:
  ```json
  {
    "success": true,
    "message": "Pengumuman hasil seleksi berhasil dipublikasikan dan tagihan daftar ulang telah diproses",
    "data": {
      "id": "a1000000-0000-0000-0000-000000000001",
      "period_id": "b0000000-0000-0000-0000-000000000001",
      "title": "Pengumuman Hasil Seleksi PMB Gelombang 1 TA 2026/2027",
      "published_at": "2026-11-12T10:00:00Z"
    }
  }
  ```

---

### 13.4 Endpoint Calon Mahasiswa (`/api/v1/selection/me`)
Akses: Role `calon_mahasiswa`, `admin_pmb`, `superadmin`.

#### 1. Lihat Hasil Seleksi & Tagihan: `GET /api/v1/selection/me/result`
Menampilkan hasil seleksi resmi bagi calon mahasiswa yang bersangkutan.
- **Sebelum Pengumuman Dirilis (Error `422 Unprocessable Entity`)**:
  ```json
  {
    "success": false,
    "message": "RESULT_NOT_PUBLISHED: pengumuman hasil seleksi belum dipublikasikan oleh panitia",
    "error_code": "RESULT_NOT_PUBLISHED"
  }
  ```
- **Setelah Pengumuman Dirilis (`200 OK`)**:
  ```json
  {
    "success": true,
    "message": "Hasil seleksi penerimaan mahasiswa baru berhasil diambil",
    "data": {
      "application_id": "a0000000-0000-0000-0000-000000000001",
      "registration_number": "PMB-2026-0001",
      "full_name": "Budi Santoso",
      "decision": "pass",
      "accepted_program": "TIF",
      "announced_at": "2026-11-12T10:00:00Z",
      "invoice_id": "i0000000-0000-0000-0000-000000000001",
      "invoice_number": "INV-REG-2026-0001",
      "amount": 5000000,
      "payment_status": "unpaid"
    }
  }
  ```

---

### 13.5 Endpoint Internal Antarlayanan (`/internal/...`)
Khusus dipanggil oleh service downstream (mis. Student Service) dengan menyertakan header `X-Internal-Token`.

#### 1. Periksa Hasil Penetapan Seleksi: `GET /internal/results/:application_id`
- **Response `200 OK`**:
  ```json
  {
    "success": true,
    "message": "Hasil seleksi berhasil diambil",
    "data": {
      "id": "r0000000-0000-0000-0000-000000000001",
      "application_id": "a0000000-0000-0000-0000-000000000001",
      "user_id": "u0000000-0000-0000-0000-000000000001",
      "decision": "pass",
      "accepted_program": "TIF",
      "announced_at": "2026-11-12T10:00:00Z"
    }
  }
  ```

---

## 14. Student Service (`pmb-student-service`)

Student Service mengelola seluruh alur pendaftaran ulang calon mahasiswa baru yang telah dinyatakan lulus seleksi, verifikasi prasyarat kliring pembayaran tagihan daftar ulang (lunas penuh atau lunas termin pertama cicilan), pengisian data atribut kemahasiswaan (ukuran jas almamater, kontak darurat, dan pernyataan kepatuhan akademik), pencatatan entitas mahasiswa baru ke tabel `students`, persiapan outbox event untuk penerbitan NIM di Tahap 11, serta rekapitulasi data dan ekspor spreadsheet Excel `.xlsx` dengan pembatasan hak akses berbasis program studi.

### 14.1 Endpoint Calon Mahasiswa (`/api/v1/student/me/...`)
Seluruh endpoint di bawah mewajibkan token JWT calon mahasiswa yang valid (`auth: required`, role: `calon_mahasiswa`).

#### 1. `GET /api/v1/student/me/reregistration`
Mengambil status dan rincian formulir pendaftaran ulang calon mahasiswa yang login.
- **Validasi Prasyarat**: Jika calon mahasiswa belum dinyatakan lulus seleksi atau pendaftaran belum terverifikasi panitia, permintaan ditolak seketika dengan status HTTP `422 Unprocessable Entity` (`NOT_PASSED`).
- **Response `200 OK`**:
  ```json
  {
    "success": true,
    "message": "Data pendaftaran ulang berhasil diambil",
    "data": {
      "reregistration": {
        "id": "e0000000-0000-0000-0000-000000000001",
        "application_id": "a0000000-0000-0000-0000-000000000001",
        "user_id": "u0000000-0000-0000-0000-000000000001",
        "accepted_program": "TI",
        "status": "waiting_payment", // waiting_payment, in_progress, completed
        "jacket_size": "XL",
        "emergency_contact_name": "Bapak Sudirman",
        "emergency_contact_phone": "081234567890",
        "statement_agreed": false,
        "completed_at": null
      },
      "is_payment_cleared": true,
      "payment_status": "partially_paid", // paid / partially_paid / unpaid
      "invoice_id": "i0000000-0000-0000-0000-000000000001",
      "can_complete": false
    }
  }
  ```

#### 2. `PUT /api/v1/student/me/reregistration`
Memperbarui data atribut registrasi mahasiswa baru (ukuran jas almamater, kontak darurat, dan pernyataan kepatuhan).
- **Request Body**:
  ```json
  {
    "jacket_size": "XL",
    "emergency_contact_name": "Bapak Sudirman",
    "emergency_contact_phone": "081234567890",
    "statement_agreed": true
  }
  ```
- **Validasi & Error**:
  - `422 Unprocessable Entity (NOT_PASSED)`: Calon mahasiswa belum dinyatakan lulus seleksi.
  - `422 Unprocessable Entity (ALREADY_COMPLETED)`: Pendaftaran ulang sudah berstatus selesai dan terkunci.
  - `422 Unprocessable Entity (VALIDATION_ERROR)`: Ukuran jas almamater tidak valid (Pilihan: `S`, `M`, `L`, `XL`, `XXL`, `XXXL`).
- **Response `200 OK`**: Mengembalikan objek detail pendaftaran ulang yang telah diperbarui dengan status transisi ke `in_progress`.

#### 3. `POST /api/v1/student/me/reregistration/complete`
Menyelesaikan alur pendaftaran ulang secara final.
- **Validasi Transaksional**:
  1. `422 Unprocessable Entity (PAYMENT_REQUIRED)`: Tagihan biaya daftar ulang belum lunas atau cicilan termin 1 belum dibayar.
  2. `422 Unprocessable Entity (INCOMPLETE_DATA)`: Data jas almamater, kontak darurat, atau pernyataan kepatuhan belum lengkap/disetujui.
- **Efek Transaksional**:
  - Status pendaftaran ulang berubah menjadi `completed`, `completed_at` dicatat waktu sekarang.
  - Entitas mahasiswa baru tercatat di tabel `students` dengan status awal `reregistered`.
  - Event outbox `nim_issuance_requested` dicatat pada tabel `outbox_events` untuk persiapan konsumsi NIM Service.
  - Status pendaftaran di Admission Service dimutasi otomatis menjadi `reregistered`.
- **Response `200 OK`**:
  ```json
  {
    "success": true,
    "message": "Pendaftaran ulang berhasil diselesaikan, Anda telah resmi terdaftar sebagai mahasiswa baru",
    "data": {
      "id": "s0000000-0000-0000-0000-000000000001",
      "user_id": "u0000000-0000-0000-0000-000000000001",
      "application_id": "a0000000-0000-0000-0000-000000000001",
      "nim": "",
      "study_program_code": "TI",
      "entry_year": 2026,
      "entry_path": "Reguler",
      "status": "reregistered",
      "created_at": "2026-09-24T06:20:00Z"
    }
  }
  ```

---

### 14.2 Endpoint Administrasi, Rekapitulasi, & Ekspor (`/api/v1/student/admin/...`)

#### 1. `GET /api/v1/student/admin/students`
Menampilkan daftar mahasiswa baru yang telah menyelesaikan daftar ulang.
- **Akses**: `admin_pmb`, `superadmin`, `pimpinan`, dan `prodi`.
- **Aturan Isolasi Role `prodi`**: Pengguna dengan role `prodi` secara otomatis dibatasi hanya dapat melihat data pada program studi yang sesuai dengan `scope_id` pada tokennya. Upaya mengakses prodi lain ditolak dengan status HTTP `403 Forbidden`.
- **Query Params**: `page`, `per_page`, `study_program_code`, `status`, `entry_year`, `search`.
- **Response `200 OK`**: Mengembalikan array objek mahasiswa baru beserta metadata paginasi standar (`page`, `per_page`, `total`).

#### 2. `GET /api/v1/student/admin/students/export?format=xlsx`
Mengunduh laporan rekapitulasi data mahasiswa baru dalam format berkas spreadsheet Excel `.xlsx`.
- **Akses**: `admin_pmb`, `superadmin`, `pimpinan`, dan `prodi` (terisolasi pada prodi scope-nya).
- **Response**: Binary stream dengan header:
  - `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
  - `Content-Disposition: attachment; filename="Laporan_Mahasiswa_Baru_YYYYMMDD_HHMMSS.xlsx"`

---

### 14.3 Endpoint Internal Inter-Service (`/internal/students/...`)
Khusus dipanggil oleh layanan downstream (seperti NIM Service di Tahap 11) dengan header `X-Internal-Token`.

#### 1. `GET /internal/students/by-application/:application_id`
Mengambil data mahasiswa berdasarkan `application_id`.
- **Response `200 OK`**: Objek data `Student`.

#### 2. `PUT /internal/students/by-application/:application_id/nim`
Memperbarui nomor induk mahasiswa (NIM) dan mengubah status mahasiswa menjadi `active`.
- **Request Body**:
  ```json
  {
    "nim": "265510001"
  }
  ```
- **Response `200 OK`**:
  ```json
  {
    "success": true,
    "message": "NIM mahasiswa berhasil diperbarui",
    "data": {
      "application_id": "a0000000-0000-0000-0000-000000000001",
      "nim": "265510001",
      "status": "active"
    }
  }
  ```

#### 3. `POST /internal/outbox/process`
Memicu pemrosesan manual antrean pesan outbox penerbitan NIM (`nim_issuance_requested`) ke NIM Service.
- **Akses**: Internal (`X-Internal-Token`)
- **Response `200 OK`**:
  ```json
  {
    "success": true,
    "message": "Antrean outbox berhasil diproses",
    "data": {
      "processed_count": 1
    }
  }
  ```

---

## 15. Layanan Nomor Induk Mahasiswa (NIM Service)

Base Path Gateway: `/api/v1/nim`
Base Path Internal: `/internal/nim`

Layanan NIM bertanggung jawab atas konfigurasi format pola penomoran NIM, pencatatan sequence counter dengan penguncian pesimistik (`SELECT ... FOR UPDATE`), penerbitan NIM yang idempoten per `application_id`, dan orkestrasi integrasi pasca penerbitan (pembaruan tabel `students`, penambahan role `mahasiswa` di Auth, perubahan status `enrolled` di Admission, dan pengiriman notifikasi `nim_issued`).

---

### 15.1 Endpoint Inter-Service Internal (`/internal/nim/...`)
Khusus dipanggil oleh Student Service (via background worker outbox) dengan header `X-Internal-Token`.

#### 1. `POST /internal/nim/issue`
Menerbitkan Nomor Induk Mahasiswa secara idempoten.
- **Header**: `X-Internal-Token: secret-internal-token`
- **Request Body**:
  ```json
  {
    "application_id": "a0000000-0000-0000-0000-000000000001",
    "user_id": "u0000000-0000-0000-0000-000000000001",
    "study_program_code": "TI",
    "entry_year": 2026,
    "selection_path": "reguler"
  }
  ```
- **Response `201 Created` (Penerbitan Baru)**:
  ```json
  {
    "success": true,
    "message": "NIM berhasil diterbitkan",
    "data": {
      "nim": "265510001",
      "application_id": "a0000000-0000-0000-0000-000000000001",
      "user_id": "u0000000-0000-0000-0000-000000000001",
      "study_program_code": "TI",
      "entry_year": 2026,
      "selection_path": "reguler",
      "sequence_number": 1,
      "issued_at": "2026-09-24T10:00:00Z",
      "is_idempotent": false
    }
  }
  ```
- **Response `200 OK` (Idempoten / Sudah Pernah Diterbitkan)**:
  ```json
  {
    "success": true,
    "message": "NIM sudah pernah diterbitkan sebelumnya (idempoten)",
    "data": {
      "nim": "265510001",
      "application_id": "a0000000-0000-0000-0000-000000000001",
      "is_idempotent": true
    }
  }
  ```

#### 2. `GET /internal/nim/issuances/:application_id`
Mengambil detail penerbitan NIM berdasarkan `application_id`.
- **Response `200 OK`**: Mengembalikan objek data `NIMIssuance`.

---

### 15.2 Endpoint Administrasi (`/api/v1/nim/admin/...`)

#### 1. `GET /api/v1/nim/admin/issuances`
Melihat daftar riwayat penerbitan NIM.
- **Akses**: `admin_pmb`, `superadmin`, `pimpinan`, `prodi` (terisolasi pada prodi scope-nya).
- **Query Params**: `page`, `per_page`, `study_program_code`, `entry_year`, `search`.
- **Response `200 OK`**: Paginasi data riwayat penerbitan NIM.

#### 2. `GET /api/v1/nim/admin/formats`
Melihat daftar template format penomoran NIM.
- **Akses**: `superadmin`
- **Response `200 OK`**: Array konfigurasi format template.

#### 3. `POST /api/v1/nim/admin/formats`
Menambahkan template format NIM baru (global atau per prodi).
- **Akses**: `superadmin`
- **Request Body**:
  ```json
  {
    "study_program_code": "TI",
    "pattern": "{YY}{nim_code_prodi}{jalur_digit}{urut:4}",
    "description": "Format Khusus Sarjana Teknik Informatika",
    "is_active": true
  }
  ```
- **Response `201 Created`**: Objek data format yang dibuat.

#### 4. `PUT /api/v1/nim/admin/formats/:id`
Memperbarui pola atau status keaktifan template format NIM.
- **Akses**: `superadmin`
- **Response `200 OK`**: Objek format yang diperbarui.

---

## 16. Dashboard Statistik Pimpinan Eksekutif (UC-31)

Dashboard Statistik Pimpinan (Executive Dashboard) mengintegrasikan visibilitas lintas layanan secara terpadu untuk peran `pimpinan` (Rektor, Wakil Rektor, Dekan) dan `superadmin`. Seluruh endpoint dilindungi dengan signature gateway dan role guard:

### 16.1 Ringkasan Agregasi Statistik Pendaftaran PMB
- **Endpoint**: `GET /api/v1/admission/admin/stats`
- **Akses**: `pimpinan`, `admin_pmb`, `superadmin`, `prodi`
- **Fungsi**: Menyajikan metrik total pendaftar, sebaran status pendaftaran (`awaiting_payment`, `form_open`, `submitted`, `verified`, `passed`, `enrolled`), sebaran per jalur masuk, dan sebaran per program studi.
- **Contoh Response `200 OK`**:
  ```json
  {
    "success": true,
    "message": "Statistik pendaftaran berhasil diambil",
    "data": {
      "total_applications": 1250,
      "by_status": {
        "awaiting_payment": 150,
        "form_open": 200,
        "submitted": 100,
        "verified": 400,
        "passed": 300,
        "enrolled": 100
      },
      "by_admission_path": {
        "1": 800,
        "2": 450
      },
      "by_period": {
        "1": 1250
      },
      "by_study_program": {
        "11111111-1111-1111-1111-111111111111": 700,
        "22222222-2222-2222-2222-222222222222": 550
      }
    }
  }
  ```

### 16.2 Ringkasan Laporan Finansial & Penerimaan Pembayaran
- **Endpoint**: `GET /api/v1/finance/admin/reports/summary`
- **Akses**: `pimpinan`, `keuangan`, `superadmin`, `admin_pmb`
- **Fungsi**: Menyajikan ringkasan total tagihan, nominal tagihan terbit, total tagihan lunas/sebagian, total nominal diterima (pendapatan kas masuk), dan ringkasan cicilan.
- **Contoh Response `200 OK`**:
  ```json
  {
    "success": true,
    "message": "Laporan ringkasan keuangan berhasil diambil",
    "data": {
      "total_invoices": 850,
      "total_amount_billed": 4250000000,
      "total_amount_received": 3100000000,
      "paid_invoices_count": 620,
      "partially_paid_invoices_count": 80,
      "unpaid_invoices_count": 150,
      "installment_summary": {
        "total_active_plans": 80,
        "total_overdue_terms": 2
      }
    }
  }
  ```

### 16.3 Monitoring Rekapitulasi Seleksi & Persetujuan Kelulusan
- **Endpoint**: `GET /api/v1/scoring/program/recap` & `GET /api/v1/selection/approval/recommendations`
- **Akses**: `pimpinan`, `superadmin`
- **Fungsi**: Memantau perolehan skor nilai seleksi per program studi, ranking tertimbang, serta melakukan telaah (approve / return) atas usulan kelulusan dari program studi.

### 16.4 Monitoring Mahasiswa Baru Terdaftar & Penerbitan NIM
- **Endpoint**: `GET /api/v1/student/admin/students` & `GET /api/v1/nim/admin/issuances`
- **Akses**: `pimpinan`, `admin_pmb`, `superadmin`, `prodi`
- **Fungsi**: Memantau calon mahasiswa yang telah menyelesaikan daftar ulang hingga resmi berstatus mahasiswa aktif dengan Nomor Induk Mahasiswa (NIM) yang telah diterbitkan.

---

## 17. Endpoints Tambahan Change Requests (CR-002, CR-003, CR-004)

### 17.1 CR-002: Pembatalan Pendaftaran, Riwayat & Kelayakan Pendaftaran Baru
- **`POST /api/v1/admission/applications/me/cancel`**
  - **Akses**: `calon_mahasiswa` (pendaftar pemilik berkas)
  - **Request Body**: `{"reason": "Pindah ke perguruan tinggi lain"}` (opsional)
  - **Keterangan**: Membatalkan pendaftaran aktif sebelum biaya pendaftaran dibayarkan. Secara sinkron membatalkan tagihan di Finance Service. Jika pembayaran sudah lunas / sedang diverifikasi, menghasilkan error `409 CANCEL_NOT_ALLOWED`.
- **`GET /api/v1/admission/applications/me/history`**
  - **Akses**: `calon_mahasiswa`
  - **Keterangan**: Mengembalikan seluruh daftar pendaftaran milik pengguna (termasuk status `cancelled` dan `failed`), diurutkan dari yang terbaru.
- **`GET /api/v1/admission/applications/me/eligibility`**
  - **Akses**: `calon_mahasiswa`
  - **Response 200 OK**:
    ```json
    {
      "success": true,
      "message": "Status kelayakan pendaftaran berhasil diambil",
      "data": {
        "can_create_new": true,
        "reason": "Anda memenuhi syarat untuk membuat pendaftaran baru",
        "blocked_choices": []
      }
    }
    ```

### 17.2 CR-003: Computer Based Test (CBT)
- **`GET /api/v1/exam/time`**
  - **Akses**: Publik
  - **Fungsi**: Sinkronisasi timer clock server dengan frontend CBT.
- **`GET /api/v1/exam/cbt/me/exams`**
  - **Akses**: `calon_mahasiswa`
  - **Fungsi**: Menampilkan daftar ujian CBT yang dijadwalkan untuk peserta beserta statusnya.
- **`GET /api/v1/exam/cbt/me/exams/:id`**
  - **Akses**: `calon_mahasiswa`
  - **Fungsi**: Menampilkan halaman detail persiapan ujian (instruksi, durasi, jumlah soal, status kesiapan tombol mulai).
- **`POST /api/v1/exam/cbt/me/exams/:id/start`**
  - **Akses**: `calon_mahasiswa`
  - **Request Body**: `{"access_token": "TOKEN_SESI"}` (bila ujian mewajibkan token)
  - **Fungsi**: Memulai pengerjaan ujian, mengunci urutan soal snapshot, dan menetapkan batas waktu deadline server.
- **`GET /api/v1/exam/cbt/me/attempts/:id`**
  - **Akses**: `calon_mahasiswa`
  - **Fungsi**: Mengambil lembar soal ujian beserta opsi jawaban yang tersanitasi (kunci jawaban `is_correct` TIDAK PERNAH dikirim demi keamanan).
- **`PUT /api/v1/exam/cbt/me/attempts/:id/answers/:question_id`**
  - **Akses**: `calon_mahasiswa`
  - **Request Body**: `{"selected_option_ids": ["uuid"], "essay_text": "...", "is_flagged": false}`
  - **Fungsi**: Autosave jawaban per butir soal. Menolak penyimpanan dengan `409 ATTEMPT_TIME_UP` jika waktu pengerjaan telah lewat.
- **`POST /api/v1/exam/cbt/me/attempts/:id/submit`**
  - **Akses**: `calon_mahasiswa`
  - **Fungsi**: Mengumpulkan hasil ujian. Sistem otomatis menghitung skor objektif dan meneruskan nilai tertimbang ke Scoring Service.
- **`CRUD /api/v1/exam/admin/cbt/banks` & `/questions`**
  - **Akses**: `admin_pmb`, `superadmin`
  - **Fungsi**: Pengelolaan bank soal, butir pertanyaan, dan pilihan jawaban.
- **`PUT /api/v1/exam/examiner/cbt/answers/:id/grade`**
  - **Akses**: `dosen` (penguji sesi)
  - **Fungsi**: Penilaian manual untuk butir soal esai.

### 17.3 CR-004: Penyelarasan Frontend & Polling Status Pembayaran
- **`GET /api/v1/admission/applications/me/progress`**
  - **Akses**: `calon_mahasiswa`
  - **Fungsi**: Stepper alur pendaftaran untuk dashboard pelamar, memuat status tiap langkah (`done`, `current`, `locked`).
- **`GET /api/v1/finance/invoices/me/:id/payments/:payment_id/status`**
  - **Akses**: `calon_mahasiswa`
  - **Fungsi**: Polling status pembayaran realtime setelah kembali dari portal Duitku sandbox/production.






