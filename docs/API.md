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


