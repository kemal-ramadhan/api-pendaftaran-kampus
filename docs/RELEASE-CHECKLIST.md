# Checklist Rilis PMB Siap Produksi (Pre-Flight Checklist)

Dokumen ini memuat daftar periksa terpadu yang **wajib dipenuhi 100%** sebelum sistem PMB dibuka untuk pendaftaran publik oleh calon mahasiswa (sesuai **CR-013**).

---

## 1. Basis Data & Migrasi Skema
- [ ] Seluruh migrasi tabel otomatis (`AutoMigrate`) berjalan sukses tanpa error.
- [ ] Tabel `fee_schemes` terisi data tarif resmi gelombang aktif.
- [ ] Jalur pendaftaran gratis (skema Rp 0) terdaftar dengan benar (`registration_fee = 0`).
- [ ] Seeder akun default development (superadmin default password) telah diubah ke kredensial produksi yang kuat.
- [ ] Database backup baseline awal telah dibuat dan disimpan di lokasi offsite yang aman.

## 2. Master Data Institusi & Periode Pendaftaran
- [ ] Fakultas dan Program Studi aktif sudah disesuaikan dengan kuota resmi per prodi.
- [ ] Periode PMB aktif (`is_active = true`) dengan rentang tanggal `start_date` dan `end_date` yang tepat.
- [ ] Jadwal tahapan seleksi (batas pengisian form, jadwal ujian CBT, pengumuman kelulusan) telah dikonfigurasi.

## 3. Integrasi Pembayaran (Duitku Produksi)
- [ ] Status merchant Duitku berstatus **Live / Production**.
- [ ] Kredensial `DUITKU_MERCHANT_CODE` dan `DUITKU_API_KEY` produksi terpasang di `.env`.
- [ ] Parameter `ALLOW_SANDBOX_IN_PRODUCTION=false` terverifikasi.
- [ ] URL Webhook (`/api/v1/finance/webhooks/payments/duitku`) terdaftar dan teruji menerima notifikasi dari Duitku.
- [ ] URL Return redirect diarahkan ke halaman status pembayaran frontend.
- [ ] Telah dilakukan pengujian nominal uji coba kecil (misal Rp 10.000) dan pembayaran sukses terverifikasi.
- [ ] Background worker rekonsiliasi pembayaran aktif (interval 10 menit).

## 4. Pengiriman Email (SMTP)
- [ ] Host SMTP produksi aktif (port 587 TLS / 465 SSL).
- [ ] Pengiriman email OTP pendaftaran sampai ke inbox calon mahasiswa dalam < 10 detik.
- [ ] Template email verifikasi, bukti pendaftaran, dan bukti pembayaran rapi dan bebas placeholder dev.

## 5. Keamanan & Jaringan
- [ ] Sertifikat SSL/TLS valid (grade A pada SSL Labs) pada domain `pmb.kampus.ac.id`.
- [ ] Port internal basis data (PostgreSQL 5432, Redis 6379, MinIO 9000) tidak diekspos ke internet publik.
- [ ] Rate limiting aktif pada endpoint autentikasi (`/login`, `/register`, `/verify`, `/forgot-password`).
- [ ] Header keamanan browser aktif di API Gateway (CORS ketat, X-Content-Type-Options, X-Frame-Options).

## 6. Uji Alur & Kesiapan Aplikasi
- [ ] Eksekusi `scripts/smoke-camaba.sh` sukses 100% tanpa kegagalan.
- [ ] Pendaftaran jalur reguler berbayar berfungsi dari checkout hingga status terbayar.
- [ ] Pendaftaran jalur gratis langsung `form_open` tanpa tagihan Duitku dan dapat dibatalkan mandiri.
- [ ] Seluruh dokumen pendukung calon mahasiswa dapat diunggah ke MinIO/S3 dan dapat diunduh staf verifikator.
- [ ] Verifikasi `powershell -ExecutionPolicy Bypass -File scripts/final-check.ps1 -Tahap rilis` status AMAN.
