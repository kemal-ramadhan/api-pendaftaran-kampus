# Architecture Decisions Log (DECISIONS.md)

Dokumen ini mencatat seluruh keputusan arsitektur dan teknis yang diambil selama pengembangan backend Sistem PMB (Auth Service & API Gateway).

---

## 1. Batas Tanggung Jawab Auth vs Identity/Organization Service
- **Status**: Dikonfirmasi lead, 18 Sep 2026
- **Keputusan**: Auth Service hanya mengelola akun login (email, phone, password hash, status, verifikasi), sesi, token RS256, OTP, reset password, role & scope, serta audit log autentikasi. Biodata lengkap (NIK, tempat/tanggal lahir, alamat, foto profil, data ortu) menjadi tanggung jawab Identity Service di fase berikutnya. Data fakultas, prodi, dan unit kerja menjadi tanggung jawab Organization Service.
- **Rasional**: Mencegah Auth Service membengkak menjadi "monolith serba ada" dan menjaga batas bounded-context microservice sesuai roadmap sistem kampus.
- **Implikasi**: Kolom `full_name` di Auth Service hanya berfungsi sebagai cache *display name* untuk keperluan pengiriman email dan klaim token.

---

## 2. Standardisasi Penamaan Endpoint
- **Keputusan**: Mengadopsi penamaan yang familiar dari API lama (EduPilot/Jatidiri) bila fungsinya sama: `POST /auth/verify`, `PUT /auth/profile`, `PUT /auth/password`, `GET /auth/health`.
- **Rasional**: Mempermudah integrasi dan transisi bagi tim frontend.
- **Implikasi**: Perilaku internal dan skema respons tetap mengikuti spesifikasi keamanan dan standar baru PRD.

---

## 3. Format Respons & Error Standar
- **Status**: Dikonfirmasi lead, 18 Sep 2026
- **Keputusan**: Menggunakan format respons JSON seragam di seluruh service:
  - Sukses: `{ "success": true, "message": "...", "data": {...}, "meta": {...} }`
  - Gagal: `{ "success": false, "message": "...", "error_code": "...", "errors": {...}, "request_id": "..." }`
- **Rasional**: Standar respons lama tidak konsisten. Standar baru menyediakan error code stabil untuk kontrol UI frontend serta request ID untuk penelusuran log.
- **Implikasi**: Diimplementasikan terpusat di `pkg/response` dan `pkg/requestid`.

---

## 4. Alur Verifikasi OTP
- **Keputusan**: Endpoint `POST /auth/verify` bersifat publik dan menerima `{ "email": "...", "otp": 123456 }` tanpa memerlukan Bearer token sebelumnya.
- **Rasional**: Pada alur pendaftaran, akun yang belum terverifikasi (`status: pending`) belum diizinkan memiliki access token aktif. Verifikasi yang berhasil akan langsung menerbitkan token (auto-login).

---

## 5. Multi-Module Go & Shared Package (`pkg`)
- **Keputusan**: 
  - Direktori `pkg/` dijadikan Go module terpisah (`github.com/pkm-asia/pmb-backend/pkg`).
  - Masing-masing service (`services/auth-service` dan `services/api-gateway`) memiliki `go.mod` sendiri dan menggunakan directive `replace github.com/pkm-asia/pmb-backend/pkg => ../../pkg`.
  - Ditambahkan `go.work` di root repository untuk kenyamanan development lokal.
  - Dockerfile tiap service menggunakan root repository sebagai build context agar package `pkg/` ikut tersalin saat container build.
- **Rasional**: Memungkinkan build berjalan mandiri di CI/CD/Docker tanpa ketergantungan mutlak pada `go.work`, namun tetap memberikan kemudahan navigasi kode di editor lokal.

---

## 6. Desain Token RS256, Rotasi Refresh, & Session ID (SID)
- **Keputusan**:
  - Token ditandatangani menggunakan algoritma asimetris RS256. Private key dipegang oleh Auth Service, dan public key dipublikasikan via JWKS (`/.well-known/jwks.json`).
  - Refresh token mendukung multi-device via penyimpanan Redis `auth:refresh:{sub}:{sid}` -> `jti` dan menerapkan rotasi JTI serta deteksi pencurian token (reuse detection).
  - Claim `sid` (session ID) **juga dimasukkan ke dalam access token** selain ke refresh token.
  - Endpoint `POST /auth/logout` cukup menggunakan Bearer token pada header `Authorization` (server mengekstrak `sid` dari access token untuk menghapus sesi di Redis). Pengiriman body `{ "refresh_token": "..." }` bersifat opsional.
  - Access token yang di-logout dimasukkan ke denylist Redis `auth:denylist:{jti}` sampai masa kedaluwarsanya habis.
- **Rasional**: Menghilangkan kelemahan token simetris tunggal pada referensi lama, mendukung logout multi-device yang aman, dan memudahkan frontend saat logout tanpa harus selalu mengirim refresh token di body.

---

## 7. Pemetaan Klaim `scopes` pada Token
- **Keputusan**: Klaim `scopes` di JWT menggunakan array of objects dengan struktur `[{"role": "prodi", "type": "prodi", "id": "TI"}]` sesuai PRD 5.2, yang dimapping dari kolom database `scope_type` dan `scope_id` pada tabel `user_roles`. API Gateway meneruskan klaim ini ke downstream via header `X-User-Scopes` sebagai JSON yang di-base64url encode.
- **Rasional**: Format ringkas dan ekspresif untuk validasi otorisasi unit kerja di service hilir.

---

## 8. Lokasi Dokumen Bisnis
- **Keputusan**: Memindahkan file `flowchart-sistem-pmb.pdf` dan `use-case-sistem-pmb.pdf` ke dalam folder `docs/`.
- **Rasional**: Menjaga kebersihan root direktori monorepo dan menyelaraskan dengan dokumentasi path di PRD.

---

## 9. Penundaan Fitur Non-Kritis ke Milestone M4
- **Status**: Disetujui lead, 18 Sep 2026
- **Keputusan**: Fitur import CSV user (`POST /auth/admin/users/import`), profil observability (Jaeger/Prometheus), dan file Postman collection ditunda pengerjaannya ke Milestone M4.
- **Rasional**: Memaksimalkan fokus dan kecepatan implementasi alur kritis calon mahasiswa berjalan penuh melalui API Gateway (M1, M2, dan M3).
- **Implikasi**: Admin API untuk manajemen user & role manual tetap diimplementasikan di M2.

