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

## 9. Status Import CSV dan Observability pada M4
- **Status**: Diputuskan pada Milestone M4, 19 Sep 2026
- **Keputusan**:
  - **Import CSV Pengguna**: Penambahan endpoint `POST /auth/admin/users/import` digeser ke fase perluasan administrasi berikutnya. Saat ini superadmin telah dilengkapi fungsionalitas manajemen pengguna lengkap per-akun (`POST /api/v1/auth/admin/users`, `PUT /users/:id/roles`, aktivasi/deaktivasi) yang telah lolos uji RBAC di gateway dan unit testing.
  - **Observability Profile**: Container `jaeger` dan `prometheus` telah didefinisikan lengkap di `deploy/docker-compose.yml` di bawah profile Docker Compose `observability`. Ekspor trace/metrics aktif saat flag atau env `OTEL_EXPORTER_ENDPOINT` dikonfigurasi.
- **Rasional**: Sesuai arahan PRD Bagian 11 dan toleransi jadwal M4: memprioritaskan hardening alur calon mahasiswa, keamanan gateway (lockout, rate-limit, anti-spoofing, token reuse), verifikasi E2E otomatis, dan kelengkapan dokumentasi frontend serta Postman collection.

---

## 10. Akses Swagger UI di Lingkungan Development
- **Keputusan**: Swagger UI auth service dibuka langsung di port host `8080` (`http://localhost:8080/docs/index.html`) pada `deploy/docker-compose.yml`, sementara API Gateway tetap memblokir path `/docs` dari akses publik (HTTP 404).
- **Rasional**: Memenuhi kebutuhan developer frontend untuk meninjau dokumentasi OpenAPI interaktif, tanpa mengorbankan aturan keamanan gateway yang melarang eksposur endpoint dokumentasi internal ke publik di luar downstream.

---

## 11. Normalisasi Line-Ending Windows dan Optimasi Build Docker
- **Keputusan**:
  - Menambahkan `.gitattributes` dengan aturan `*.sh text eol=lf` agar script shell tidak terkonversi menjadi format CRLF di sistem operasi Windows.
  - Menambahkan `.dockerignore` di root repositori untuk mengabaikan folder `reference/`, `docs/`, `scripts/`, `.git/`, dan file `.pdf`/`.md` dari context Docker build.
- **Rasional**: Mencegah script shell rusak di Windows saat dieksekusi dengan Git Bash, serta memangkas ukuran build context dari 48 MB menjadi sangat ringkas sehingga proses `docker build` selesai dalam hitungan detik.

---

## 12. Fase 2: Shared Foundation (F2-M0) — Paket Bersama & Outbox Pattern
- **Status**: Dikonfirmasi dan disetujui lead, 21 Sep 2026
- **Keputusan**:
  - Menyediakan paket bersama di `pkg/`:
    - `pkg/internalclient`: HTTP client terstandarisasi untuk panggilan `/internal/*` dengan timeout 5s, auto-retry 2x untuk GET, dan propagasi `X-Internal-Token` serta `X-Request-ID`.
    - `pkg/storage`: Klien MinIO/S3 dengan sniffer deteksi MIME type 512 byte pertama (`http.DetectContentType`) dan generator presigned URL.
    - `pkg/pagination`: Helper limit-offset, sorting aman, dan mapping metadata paginasi standar.
    - `pkg/outbox`: Implementasi Outbox pattern (`outbox_events` tabel, repository, dan worker poller) untuk menjamin propagasi status mutasi lintas service tanpa inkonsistensi data.
    - `pkg/gatewayauth`: Helper ekstraksi identitas (`GetUserID`, `GetUserRoles`, `GetUserScopes`, `RequireRole`).
    - `pkg/internalauth`: Ekstraksi middleware verifikasi token internal bersama.
- **Rasional**: Memastikan 10 microservice baru di Fase 2 dibangun di atas fondasi yang konsisten tanpa menduplikasi kode infrastruktur.

---

## 13. Standar URL Webhook Payment Provider di Gateway
- **Keputusan**: URL publik webhook payment gateway diarahkan ke `/api/v1/finance/webhooks/payments/:provider` dengan mode `auth: public` pada gateway, lalu di-rewrite ke Finance Service.
- **Rasional**: Menjaga konsistensi namespace gateway (`/api/v1/<service>/*`) tanpa melanggar batasan publik provider.

---

## 14. Pemisahan Bucket MinIO per Domain Layanan
- **Keputusan**: Berkas disimpan dalam bucket privat terpisah per service:
  - `pmb-admission-documents`
  - `pmb-identity-photos`
  - `pmb-finance-proofs`
  - `pmb-finance-receipts`
  - `pmb-exam-cards`
  Seluruh berkas bersifat privat dan hanya dapat diakses melalui presigned URL berumur 5 menit.
- **Rasional**: Mencegah kebocoran berkas antar domain dan membatasi blast radius hak akses storage.

---

## 15. Pewarisan Batas Body (MaxBody) Gateway dari Service ke Route
- **Keputusan**: Gateway `LoadRoutesConfig` mewarisi `MaxBody` dari `services[name].MaxBody` jika route tidak menetapkan `max_body` spesifik. Default gateway adalah 1 MB, sedangkan route unggah dokumen diset 5–10 MB.
- **Rasional**: Mencegah request multipart upload foto profil dan dokumen pendaftaran tertolak oleh batas default gateway 1 MB.

---

## 16. Format Enkripsi NIK At-Rest dan Hashing Unik
- **Keputusan**: NIK disimpan dalam format AES-256-GCM (string base64 gabungan nonce 12 byte + ciphertext). Kolom `nik_hash` menyimpan SHA-256 hex dari NIK polos dengan indeks UNIQUE di database untuk verifikasi keunikan tanpa mendeskripsi seluruh isi tabel.
- **Rasional**: Memenuhi standar privasi data pribadi (UU PDP) sekaligus menjaga performa pencarian unik di database.

---

## 17. Fase 2: OCR Service KTP & Peningkatan Kualitas Gambar Non-Generatif (F2-M2b)
- **Status**: Diputuskan pada Milestone F2-M2b, 21 Sep 2026
- **Keputusan**:
  - OCR Service dibangun menggunakan **Python 3.12 + FastAPI**, berjalan di Docker container berbasis `python:3.12-slim` dengan Tesseract OCR (`ind+eng`), OpenCV headless, Pillow, dan Redis.
  - Menjaga kepatuhan kontrak Fase 1: middleware verifikasi `X-Internal-Token`, propagasi `X-Request-ID`, format response JSON standar (`success`, `message`, `data`/`error_code`), serta endpoint healthcheck (`/health` dan `/health/ready`).
  - **Larangan Generative AI**: Peningkatan kualitas gambar diwajibkan 100% menggunakan teknik pengawetan konten (*content-preserving*): deskewing & perspective warp (OpenCV), Bilateral Filtering (denoise), CLAHE pada kanal Luminance, Unsharp Masking, dan Lanczos non-generative scaling. Tidak menggunakan model generatif (GAN/diffusion) agar tidak mengarang atau mengubah karakter NIK/teks KTP.
  - **Kepatuhan Privasi Data Pribadi (UU No. 27/2022)**: Parameter `OCR_CLOUD_ENABLED=false` aktif secara default, tidak ada data yang dikirim ke penyedia cloud luar, data hasil OCR hanya disimpan sementara di Redis dengan TTL 15 menit, dan log audit tidak memuat NIK atau file citra.
  - **Pengujian Tanpa KTP Asli**: Seluruh fixture pengujian dan benchmark dihasilkan menggunakan generator KTP sintetis buatan (`services/ocr-service/tests/synthetic_ktp_generator.py`) dengan data fiktif.
  - **Benchmark**: Dilengkapi perintah `make ocr-benchmark` yang mengeksekusi evaluasi performa dan akurasi terotomasi ke `docs/ocr-benchmark.md`.
- **Rasional**: Menghadirkan kapabilitas OCR yang cepat (rata-rata 2.10 detik di CPU), akurat (NIK exact match 100% pada gambar jernih & buram ringan), aman dari risiko halusinasi AI generatif, dan patuh regulasi privasi data.

---

## 18. Fase 2: Organization Service & Validasi Scope Prodi di Auth Service (F2-M1)
- **Status**: Diputuskan pada Tahap 1, 23 Sep 2026
- **Keputusan**:
  - Organization Service dibangun sebagai pemilik tunggal master data fakultas dan program studi di database `pmb_organization` (`services/organization-service`).
  - Endpoint publik (`GET /organization/faculties`, `GET /organization/study-programs`, `GET /organization/study-programs/:code`) dilindungi middleware `pkg/gatewayauth` untuk memvalidasi tanda tangan HMAC API Gateway.
  - Endpoint manajemen administratif (`/organization/admin/*`) dibatasi ketat hanya untuk peran `superadmin`.
  - Endpoint internal `/internal/*` dilindungi middleware `pkg/internalauth` dengan header `X-Internal-Token`.
  - Sesuai PRD Bagian 4.10, Auth Service memanfaatkan `pkg/internalclient` untuk memvalidasi `scope_id` saat superadmin menetapkan peran `prodi` (`CreateUser` dan `UpdateUserRoles`). Jika kode prodi tidak ada di Organization Service, permintaan ditolak dengan HTTP 422 (`VALIDATION_ERROR`). Jika Organization Service tidak dapat dijangkau, permintaan ditolak dengan HTTP 502 (`UPSTREAM_UNAVAILABLE`).
- **Rasional**: Mencegah pendaftaran role prodi dengan scope fiktif di basis data Auth, menjaga konsistensi master data kampus, dan memberikan proteksi kegagalan jaringan yang transparan ke klien.

---

## 19. Fase 2: Identity Service & Integrasi Asinkron OCR KTP (F2-M2 & F2-M2b)
- **Status**: Diputuskan pada Tahap 2, 23 Sep 2026
- **Keputusan**:
  - Identity Service dibangun sebagai pengelola tunggal data personal calon mahasiswa (`services/identity-service`) pada basis data `pmb_identity`.
  - **Enkripsi NIK At-Rest**: NIK 16 digit wajib disimpan terenkripsi menggunakan AES-256-GCM dengan kunci simetris dari environment `IDENTITY_ENCRYPTION_KEY`. Kolom `nik_hash` (SHA-256) berindeks UNIQUE digunakan untuk validasi keunikan tanpa mendekripsi tabel secara massal.
  - **Masking NIK Berbasis Hak Akses**: NIK hanya dikembalikan utuh ke pemilik data (`user_id`), role `admin_pmb`/`superadmin`, dan pemanggilan internal service (`/internal/*`). Untuk pihak lain, NIK otomatis dimasking (`3205********0001`).
  - **Alur Pemindaian OCR KTP**:
    1. Pengguna mengunggah gambar KTP ke `POST /api/v1/identity/me/ktp-scan`.
    2. Gambar asli disimpan ke MinIO bucket privat `pmb-identity-ktp` dan diteruskan ke OCR Service via `POST /internal/ocr/ktp`.
    3. Status dipantau secara asinkron via `GET /api/v1/identity/me/ktp-scan/:scan_id`.
    4. Pengguna wajib mengonfirmasi data hasil pembacaan via `POST /api/v1/identity/me/ktp-scan/:scan_id/confirm`. Seluruh modifikasi field yang dilakukan user dicatat di `fields_edited_by_user` untuk audit verifikator admin. Data mentah OCR tidak disimpan permanen di database.
  - **Penyimpanan Berkas**: Foto profil disimpan di bucket `pmb-identity-photos` dan gambar KTP disimpan di `pmb-identity-ktp` (akses publik langsung ditutup, hanya lewat presigned URL berumur terbatas).
  - **Evaluasi Kelengkapan Biodata**: Endpoint `/me/completeness` dan `/internal/persons/:user_id/completeness` mengevaluasi 4 domain wajib (profil, alamat KTP, minimal 1 orang tua/wali, dan riwayat pendidikan terakhir) sebelum pendaftaran dapat di-submit ke Admission Service.
- **Rasional**: Menjamin kepatuhan UU Pelindungan Data Pribadi (UU No. 27/2022), mencegah kebocoran NIK/foto, mengotomasi pengisian form calon mahasiswa tanpa risiko salah baca dari AI, serta menyediakan data yang terverifikasi bagi tahapan admission.

---

## 20. Fase 2: Notification Service & Admission Service Bagian 1 (F2-M3 & F2-M4)
- **Status**: Diputuskan pada Tahap 3, 23 Sep 2026
- **Keputusan**:
  - **Notification Service Terpusat (Internal Service)**: Dibangun sebagai layanan internal murni di port 8083 (`services/notification-service`) dan basis data `pmb_notification`. Tidak diekspos melalui API Gateway publik melainkan dipanggil oleh service lain via `POST /internal/notifications` dilindungi `pkg/internalauth`.
  - **Driver & Antrean Worker**: Driver email menggunakan SMTP standard library (default Mailpit port 1025). Driver WhatsApp menggunakan driver terstruktur `log`. Antrean worker latar belakang berjalan periodik untuk me-retry pengiriman yang gagal hingga maksimal 3 kali.
  - **Format Nomor Registrasi**: Nomor registrasi pendaftaran dirancang dengan pola `PMB-{YYYY}-{period_id}-{urut 5 digit}` (contoh `PMB-2026-1-00001`), unik per periode.
  - **State Machine Pendaftaran Bagian 1**: Pendaftaran baru otomatis masuk ke status `awaiting_payment`. Pilihan program studi (choice_1, choice_2) hanya dapat diubah saat berstatus `draft` atau `awaiting_payment`; upaya perubahan pada status setelah verifikasi pembayaran ditolak dengan error `INVALID_STATE`. Pendaftaran di luar rentang periode ditolak dengan `PERIOD_CLOSED`, dan pendaftaran berulang user pada periode yang sama ditolak dengan `DUPLICATE_APPLICATION`.
  - **Transactional Outbox ke Finance**: Sesuai PRD Bagian 4.4, pembuatan tagihan pendaftaran menggunakan pola Transactional Outbox (`pkg/outbox`) ke endpoint Finance `POST /internal/invoices`. Karena Finance Service baru dibuat pada Tahap 4, event outbox tersimpan secara andal dalam transaksi database `pmb_admission` dan dibiarkan tertahan (retry) oleh worker outbox tanpa menggagalkan proses pembuatan formulir pendaftar.
  - **Isolasi Akses Role Prodi**: Staf program studi yang mengakses `/api/v1/admission/admin/applications` hanya diperbolehkan melihat pendaftar yang memilih prodinya sebagai pilihan 1 atau pilihan 2 berdasarkan scope prodi yang melekat pada token/claim pengguna.
- **Rasional**: Memisahkan antrean notifikasi dari dependensi pihak ketiga, menjamin konsistensi data pendaftaran dan penagihan dengan jaminan eventual consistency outbox pattern, serta menjaga isolasi data antar prodi sesuai prinsip least privilege.

---

## 21. Fase 2: Finance Service Bagian 1 (F2-M5)
- **Status**: Diputuskan pada Tahap 4, 23 Sep 2026
- **Keputusan**:
  - **Arsitektur Finance Service**: Dibangun sebagai microservice mandiri pada port 8085 (`services/finance-service`) dengan basis data Postgres `pmb_finance`.
  - **Integritas Nominal Integer Rupiah**: Semua nilai nominal uang (`amount`, `paid_amount`) wajib menggunakan tipe `int64` di Go dan `BIGINT` di Postgres untuk mengeliminasi potensi distorsi pembulatan bilangan desimal/float.
  - **Eventual Consistency Outbox Admission -> Finance**:
    - Saat pendaftaran dibuat di Admission Service, event outbox `invoice.create` dikirimkan oleh background outbox worker ke endpoint Finance `POST /internal/invoices`.
    - Finance Service membentuk invoice resmi dengan format nomor `INV-YYYYMMDD-XXXXX` berstatus `unpaid`.
  - **Penyelarasan Status Pendaftaran (form_open)**:
    - Ketika tagihan pendaftaran dinyatakan lunas (baik via Virtual Account maupun transfer manual yang disetujui), Finance Service memanggil Admission Service melalui `POST /internal/applications/:id/status` dengan payload `{ "status": "form_open" }`.
    - Sekaligus mengirim notifikasi konfirmasi pembayaran lunas ke Notification Service via `POST /internal/notifications`.
  - **Payment Provider & Dev Simulate**:
    - Komunikasi gateway pembayaran diisolasi melalui antarmuka `PaymentProvider`. Pada tahap ini diimplementasikan `MockProvider` yang menerbitkan nomor Virtual Account sintetis.
    - Untuk mempermudah otomasi pengujian dan e2e tanpa payment gateway eksternal, disediakan endpoint `POST /dev/payments/:id/simulate-paid` yang dijaga ketat dengan guard `cfg.IsDevelopment()`. Pada environment staging/production, endpoint ini otomatis dinonaktifkan (HTTP 403/404).
  - **Idempotensi Webhook Provider**:
    - Webhook provider diproses melalui `POST /webhooks/payments/:provider`.
    - Setiap webhook dicatat pada tabel `payment_webhook_logs`. Jika event dengan `provider_ref` yang sama dikirim ulang, sistem mendeteksi duplikasi secara idempoten dan mengembalikan respons sukses tanpa memproses ulang pelunasan.
    - Header `X-Provider-Signature` diverifikasi dengan HMAC SHA256; jika signature tidak valid, request langsung ditolak dengan HTTP 401.
  - **Penerbitan Kwitansi Elektronik PDF**:
    - Kwitansi resmi diterbitkan secara otomatis saat pembayaran berhasil diverifikasi dengan nomor kwitansi `KW-YYYYMMDD-XXXXX`.
    - Berkas PDF kwitansi di-generate menggunakan generator PDF 1.4 murni standar library Go tanpa ketergantungan library CGO eksternal, lalu diunggah ke MinIO bucket `pmb-finance-receipts` dan dapat diunduh melalui `GET /api/v1/receipts/me/:payment_id`.
  - **Alur Verifikasi Transfer Manual**:
    - Calon mahasiswa dapat mengunggah bukti transfer manual (`multipart/form-data`) ke `POST /api/v1/invoices/me/:id/manual-transfer` yang disimpan di MinIO bucket `pmb-finance-proofs` dengan status awal `awaiting_verification`.
    - Staf keuangan melihat antrean via `GET /api/v1/finance/admin/payments?status=awaiting_verification` dan melakukan tindakan `approve` (melunasi tagihan & menerbitkan kwitansi) atau `reject` (calon mahasiswa diizinkan mengunggah ulang bukti baru).
- **Rasional**: Mencegah kesalahan kalkulasi uang melalui pembulatan, menjamin isolasi dan keamanan data finansial calon mahasiswa, memastikan idempotenitas transaksi, serta mengintegrasikan alur pendaftaran dan pembayaran secara otomatis dan reliabel.

---

## 22. Fase 2: Admission Service Bagian 2 — Dokumen Persyaratan, Magic Bytes Sniffing, Submit Pendaftaran, Verifikasi Admin, dan Ekspor Excel (F2-M6)
- **Status**: Diputuskan pada Tahap 5, 23 Sep 2026
- **Keputusan**:
  - **Validasi Keamanan Magic Bytes Berkas Persyaratan**:
    - Seluruh unggahan dokumen pendaftaran (`POST /api/v1/admission/applications/me/documents`) divalidasi tipe konten sebenarnya menggunakan `pkg/storage.SniffAndValidate` (membaca 512 byte pertama stream berkas). Berkas executable atau script biner yang berkedok ekstensi `.pdf` (mis. `virus.pdf` dengan magic bytes `MZ`) ditolak secara dini dengan HTTP 415 (`FILE_TYPE_NOT_ALLOWED`).
    - Berkas yang lolos disimpan ke bucket MinIO privat `pmb-admission-documents` dengan object key berstruktur `documents/{appID}/{docType}_{timestamp}_{filename}`.
  - **Validasi Ketat Sebelum Submit Pendaftaran**:
    - Pendaftar hanya dapat melakukan submit (`POST /api/v1/admission/applications/me/submit`) saat status berada pada `form_open` atau `revision_required`.
    - **Pemeriksaan Kelengkapan Biodata**: Admission Service memanggil Identity Service (`GET /internal/persons/:user_id/completeness`). Jika biodata (profil, alamat KTP, wali, riwayat pendidikan) belum lengkap, pendaftaran ditolak dengan kode error `BIODATA_INCOMPLETE`.
    - **Pemeriksaan Berkas Dokumen Wajib**: Seluruh dokumen dengan `is_required = true` pada jalur pendaftaran terkait wajib telah terunggah dan tidak dalam status `invalid`. Jika ada dokumen wajib yang kurang, pendaftaran ditolak dengan kode error `DOCUMENT_REQUIRED`.
    - Setelah lolos, status bertransisi ke `submitted` dan sistem mengirimkan notifikasi konfirmasi email (`application_submitted`).
  - **Alur Verifikasi Dokumen & Mekanisme Revisi**:
    - Admin PMB memverifikasi berkas pendaftar via `POST /api/v1/admission/admin/applications/:id/verify` dengan payload evaluasi per berkas (`valid` atau `invalid` beserta catatan perbaikan).
    - Jika terdapat minimal 1 dokumen yang dinyatakan `invalid`, status pendaftaran otomatis beralih ke `revision_required` dan sistem mengirimkan notifikasi catatan revisi (`application_revision_required`). Calon mahasiswa kemudian mengunggah ulang dokumen perbaikan dan melakukan submit ulang.
    - Jika seluruh dokumen wajib telah berstatus `valid`, status pendaftaran otomatis beralih ke `verified` dan sistem mengirimkan notifikasi kelulusan berkas (`application_verified`).
  - **Presigned URL Dokumen Admin 5 Menit**:
    - Khusus untuk akses admin/panitia PMB (`GET /api/v1/admission/admin/applications/:id/documents/:code/url`), presigned URL MinIO disetel kedaluwarsa tepat 300 detik (5 menit) untuk memitigasi risiko kebocoran tautan dokumen privat calon mahasiswa.
  - **Ekspor Data Pendaftar Excel Spreadsheet (.xlsx)**:
    - Admin PMB dapat mengekspor daftar pendaftar lengkap dengan seluruh metadata pilihan prodi dan status pendaftaran ke format `.xlsx` via `GET /api/v1/admission/admin/applications/export` menggunakan library `github.com/xuri/excelize/v2` dengan styling header profesional, lebar kolom otomatis, dan format tanggal waktu standar.
  - **Agregasi Statistik Pendaftaran Real-Time**:
    - Endpoint `GET /api/v1/admission/admin/stats` menyajikan ringkasan statistik komprehensif: total pendaftar, distribusi per status (`awaiting_payment`, `form_open`, `submitted`, `revision_required`, `verified`), sebaran per program studi, sebaran per jalur masuk, dan per gelombang/periode.
  - **Tampilan Terpadu Detail Pendaftar Admin**:
    - Endpoint detail pendaftar `GET /api/v1/admission/admin/applications/:id` bagi panitia PMB menggabungkan data formulir (Admission), biodata lengkap dan data OCR KTP unmasked (Identity Service), serta status tagihan dan pembayaran (Finance Service) dalam satu representasi komprehensif.
- **Rasional**: Menjamin integritas berkas calon mahasiswa dari ancaman malware berkas palsu, memastikan kepatuhan alur verifikasi berkas sebelum seleksi, memberikan kemudahan audit administratif bagi panitia PMB, dan menjaga keamanan dokumen privat dengan masa berlaku presigned URL yang ketat.




