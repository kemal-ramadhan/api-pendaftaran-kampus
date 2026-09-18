# Kontrak API — Sistem PMB (Fase 1: Auth & Gateway)

Dokumen ini adalah panduan integrasi API untuk tim frontend dan service hilir. Semua request publik diarahkan melalui **API Gateway** pada base URL:

```
http://localhost:8000/api/v1
```

---

## 1. Standar Format Respons

### A. Format Respons Sukses
```json
{
  "success": true,
  "message": "Pesan deskriptif keberhasilan",
  "data": { ... },
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 120
  }
}
```
*Catatan: Objek `meta` hanya disertakan pada endpoint yang mendukung paginasi.*

### B. Format Respons Error
```json
{
  "success": false,
  "message": "Pesan ramah pengguna dalam Bahasa Indonesia",
  "error_code": "INVALID_CREDENTIALS",
  "errors": {
    "email": ["Format email tidak valid"]
  },
  "request_id": "c56a4180-65aa-42ec-a945-5fd21dec0538"
}
```
*Catatan: Objek `errors` hanya muncul pada status `422 Unprocessable Entity` untuk error validasi input form.*

---

## 2. Daftar Error Code Standar

| Error Code | HTTP Status | Keterangan |
|---|---|---|
| `VALIDATION_ERROR` | 422 | Input form tidak valid (lihat rincian di field `errors`) |
| `EMAIL_TAKEN` | 409 | Email sudah terdaftar dan terverifikasi |
| `INVALID_CREDENTIALS` | 401 | Email tidak ditemukan atau kata sandi salah |
| `EMAIL_NOT_VERIFIED` | 403 | Akun belum verifikasi OTP email (frontend arahkan ke halaman OTP) |
| `ACCOUNT_INACTIVE` | 403 | Akun dinonaktifkan oleh administrator |
| `TOO_MANY_ATTEMPTS` | 429 | Gagal login 5 kali berturut-turut, akun terkunci 15 menit |
| `OTP_INVALID` | 400 | Kode OTP salah (disertai sisa percobaan `remaining_attempts`) |
| `OTP_EXPIRED` | 400 | Kode OTP telah kedaluwarsa |
| `OTP_LOCKED` | 429 | Percobaan OTP melebihi batas maksimal, wajib minta kirim ulang |
| `OTP_RESEND_COOLDOWN` | 429 | Pengiriman ulang OTP belum melewati batas cooldown 60 detik |
| `RESET_TOKEN_INVALID` | 400 | Token reset password salah, kedaluwarsa, atau sudah pernah digunakan |
| `TOKEN_MISSING` | 401 | Header `Authorization: Bearer <token>` tidak disertakan |
| `TOKEN_INVALID` | 401 | Format token tidak valid atau tanda tangan digital salah |
| `TOKEN_EXPIRED` | 401 | Access token telah kedaluwarsa (panggil `/auth/refresh`) |
| `TOKEN_REVOKED` | 401 | Token telah di-logout (masuk denylist) atau sesi dicabut |
| `FORBIDDEN` | 403 | Pengguna tidak memiliki role/hak akses yang disyaratkan |
| `NOT_FOUND` | 404 | Endpoint atau data tidak ditemukan |
| `RATE_LIMITED` | 429 | Batas kuota request IP tercapai |
| `UPSTREAM_UNAVAILABLE` | 502 | Service hilir sedang tidak dapat dihubungi |
| `UPSTREAM_TIMEOUT` | 504 | Service hilir melebihi batas waktu pemrosesan (timeout) |
| `INTERNAL_ERROR` | 500 | Kesalahan internal server tak terduga |

---

## 3. Alur Endpoint Calon Mahasiswa (via Gateway)

| Halaman / Fitur | Method | Endpoint Gateway | Auth | Catatan |
|---|---|---|---|---|
| **Register** | `POST` | `/api/v1/auth/register` | Public | Sukses → redirect ke halaman verifikasi OTP membawa `email` |
| **Verifikasi OTP** | `POST` | `/api/v1/auth/verify` | Public | Sukses → auto-login mengembalikan token, redirect ke Dashboard |
| **Kirim Ulang OTP** | `POST` | `/api/v1/auth/resend-otp` | Public | Tombol aktif setelah `resend_available_at` (cooldown 60s) |
| **Login** | `POST` | `/api/v1/auth/login` | Public | Mengembalikan `access_token` & `refresh_token` di body JSON |
| **Refresh Token** | `POST` | `/api/v1/auth/refresh` | Public | Body `{ "refresh_token": "..." }`, merotasi token |
| **Lupa Password** | `POST` | `/api/v1/auth/forgot-password` | Public | Selalu mengembalikan pesan sukses generik |
| **Reset Password** | `POST` | `/api/v1/auth/reset-password` | Public | Body `{ "token": "...", "password": "...", "password_confirmation": "..." }` |
| **Profil Saya** | `GET` | `/api/v1/auth/me` | Bearer | Mengambil data akun, status, daftar role, dan scopes |
| **Ubah Profil Akun** | `PUT` | `/api/v1/auth/profile` | Bearer | Hanya untuk mengubah `full_name` dan `phone` |
| **Ganti Password** | `PUT` | `/api/v1/auth/password` | Bearer | Body `{ "current_password": "...", "password": "..." }`, mencabut sesi lain |
| **Logout Sesi Ini** | `POST` | `/api/v1/auth/logout` | Bearer | Cukup kirim Bearer token (sid diambil dari token). Body `{ "refresh_token": "..." }` bersifat opsional |
| **Logout Semua Sesi** | `POST` | `/api/v1/auth/logout-all` | Bearer | Mencabut seluruh sesi login user di semua perangkat |

---

## 4. Migrasi dari API Auth Lama (EduPilot / Jatidiri)

| API Lama | API Baru PMB | Perubahan Perilaku & Catatan |
|---|---|---|
| `POST /auth/register` (`name`, `role`) | `POST /auth/register` | Menggunakan field `full_name`, `phone`, `password_confirmation`. Input `role` diabaikan (selalu `calon_mahasiswa`). Tidak langsung memberikan token. |
| `POST /auth/verify` (perlu Bearer token) | `POST /auth/verify` | Bersifat **publik**, body `{ "email": "...", "otp": 123456 }`. Sukses langsung menerbitkan token (auto-login). |
| `POST /auth/resend-otp` (perlu Bearer token) | `POST /auth/resend-otp` | Bersifat **publik**, body `{ "email": "..." }`, menerapkan rate-limit cooldown. |
| `POST /auth/login` | `POST /auth/login` | Pesan error generik (`INVALID_CREDENTIALS`), lockout proteksi brute force, token membawa claim `roles` dan `scopes`. |
| `POST /auth/refresh` | `POST /auth/refresh` | Menerapkan rotasi refresh token otomatis, deteksi reuse (pencurian sesi), dan dukungan multi-device. |
| `POST /auth/logout` | `POST /auth/logout` (+ `/auth/logout-all`) | Access token otomatis dimasukkan ke denylist Redis sampai kedaluwarsa. Mendukung logout sesi tunggal maupun semua sesi. |
| `GET /auth/me` | `GET /auth/me` | Data akun + `roles` + `scopes` (objek legacy `customer` dihilangkan). |
| `PUT /auth/profile` (`birth_date`, dll.) | `PUT /auth/profile` | Hanya untuk `full_name` dan `phone`. Biodata pendaftaran lainnya berpindah ke Identity Service. |
| `PUT /auth/profile/:id` | **Dihapus** | Celah IDOR ditutup. Administrator menggunakan endpoint `PUT /auth/admin/users/:id`. |
| `PUT /auth/photo-profile` | **Dihapus** | Dipindahkan ke Identity Service pada fase berikutnya. |
| `PUT /auth/change-role` | **Dihapus** | Celah privilege escalation ditutup. Role hanya diubah oleh superadmin via `PUT /auth/admin/users/:id/roles`. |
| `POST /auth/customers/import` | `POST /auth/admin/users/import` | Khusus superadmin via CSV dengan format standar. |
| `/customer/*` dan `/other/*` | **Dihapus** | Rute terekspos tanpa autentikasi dihapus. Komunikasi internal dialihkan ke `/internal/*`. |
