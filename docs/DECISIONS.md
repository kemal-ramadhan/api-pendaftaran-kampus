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

---

## 23. Fase 2: Exam Service — Sesi Ujian, Penugasan Dosen, Penjadwalan Peserta, Presensi, dan Kartu Ujian PDF dengan QR Code (F2-M7)
- **Status**: Diputuskan pada Tahap 6, 23 Sep 2026
- **Keputusan**:
  - **Arsitektur Exam Service Mandiri**:
    - Layanan Exam Service (`pmb-exam-service`) beroperasi sebagai microservice independen pada port `8087` dengan basis data relasional terisolasi `pmb_exam`.
    - Mematuhi standar keamanan `docs/AGENT_RULES.md`: verifikasi HMAC API Gateway (`pkg/gatewayauth`), otentikasi inter-service token internal (`pkg/internalauth`), migrasi skema SQL terstruktur, dan isolasi role ketat.
  - **Manajemen Sesi Ujian & Validasi Kapasitas**:
    - Mendukung tipe sesi ujian tulis (`written_test`) dan wawancara (`interview`).
    - Setiap sesi membatasi kuota peserta (`capacity`). Penjadwalan peserta baru baik secara manual maupun batch otomatis (*auto-assign*) memvalidasi kuota sesi yang tersisa; penambahan yang melebihi kapasitas ditolak dengan error `422 Unprocessable Entity (SESSION_CAPACITY_EXCEEDED)`.
  - **Prasyarat Status Verified untuk Penjadwalan**:
    - Hanya pendaftar yang telah lolos verifikasi berkas di Admission Service (`status = verified`) yang dapat dijadwalkan ke sesi ujian. Upaya penjadwalan pendaftar yang belum berstatus `verified` ditolak dengan kode `APPLICATION_NOT_VERIFIED` (HTTP 422).
    - Setelah berhasil dijadwalkan, Exam Service secara atomik memanggil Admission Service (`POST /internal/applications/:id/status`) untuk memutasi status pendaftaran ke `exam_scheduled` dan memicu notifikasi email jadwal ujian ke Notification Service.
  - **Penomoran Peserta & Penerbitan Kartu Ujian Resmi (PDF + QR Code)**:
    - Nomor peserta ujian diterbitkan unik dengan format resmi: `EXAM-{YYYY}-{urut:05d}`.
    - Generator kartu ujian memproduksi berkas standar `%PDF-1.4` murni Go (tanpa library CGO eksternal).
    - Kartu memuat: nomor peserta, nama pendaftar (dari Identity Service), pas foto, rincian jadwal, lokasi/link virtual, tata tertib, dan kode QR 2D berbasis vektor resolusi tajam (menggunakan `github.com/skip2/go-qrcode`).
    - Berkas kartu tersimpan privat di MinIO bucket `pmb-exam-cards` dan dapat diunduh oleh calon mahasiswa via `GET /api/v1/exam/me/card`.
  - **Isolasi Akses & Presensi Kehadiran Dosen Penguji**:
    - Dosen penguji (role `dosen`) hanya diizinkan melihat sesi dan peserta yang secara eksplisit ditugaskan kepada mereka (`session_examiners`).
    - Presensi kehadiran peserta (`present` / `absent`) dapat dicatat oleh Admin PMB atau oleh Dosen Penguji yang bertugas pada sesi tersebut via `PATCH /api/v1/exam/admin/participants/:id/attendance`. Upaya perubahan presensi oleh dosen pada sesi lain ditolak dengan HTTP 403 (`ErrForbiddenExaminer`).
- **Rasional**: Menjamin ketertiban seleksi penerimaan mahasiswa baru, memastikan integritas dokumen identitas peserta di ruang ujian melalui QR code verifikasi pengawas, dan mengisolasi wewenang dosen penguji secara aman sesuai pembagian tugas.

---

## 24. Fase 2: Scoring Service — Master Komponen & Bobot Nilai 100%, Otorisasi Dosen Penguji, Rata-Rata Tertimbang, Pemeringkatan Prodi Dinamis, Ekspor Excel XLSX, dan Proteksi Penguncian Nilai (F2-M8)
- **Status**: Diputuskan pada Tahap 7, 24 Sep 2026
- **Keputusan**:
  - **Arsitektur Scoring Service Mandiri**:
    - Layanan Scoring Service (`pmb-scoring-service`) beroperasi sebagai microservice independen pada port `8089` dengan basis data terisolasi `pmb_scoring`.
    - Mematuhi standar keamanan `docs/AGENT_RULES.md`: verifikasi HMAC API Gateway (`pkg/gatewayauth`), autentikasi inter-service token internal (`pkg/internalauth`), migrasi skema SQL terstruktur, dan penanganan error standar.
  - **Validasi Mutlak Total Bobot Komponen Nilai 100%**:
    - Komponen penilaian dikonfigurasikan per jalur seleksi (`path_id`).
    - Total persentase bobot seluruh komponen aktif pada satu jalur seleksi wajib tepat **100.00%**. Upaya penambahan atau perubahan bobot komponen yang menyebabkan akumulasi bobot melebihi 100% langsung ditolak dengan status HTTP `422 Unprocessable Entity (INVALID_WEIGHT_SUM)`.
  - **Otorisasi Dosen Penguji via Inter-Service Exam**:
    - Setiap kali dosen penguji (role `dosen`) menginput atau mengubah nilai peserta (`PUT /api/v1/scoring/examiner/scores/:application_id`), Scoring Service secara internal memverifikasi ke Exam Service (`GET /internal/participants/:application_id` dan `GET /internal/sessions/:session_id`).
    - Nilai hanya dapat diinput jika `examiner_user_id` dosen bersangkutan tercantum dalam daftar penguji resmi pada sesi ujian peserta tersebut. Upaya input nilai untuk peserta di luar sesi dosen ditolak dengan HTTP `403 Forbidden`. Role `admin_pmb` dan `superadmin` memiliki wewenang administratif untuk menginput nilai tanpa batasan sesi penguji.
  - **Proteksi Penguncian Nilai (Flag Locked)**:
    - Sesuai PRD Bagian 4.7, setelah kombinasi gelombang dan program studi dikunci oleh alur kelulusan (`program_locks.is_locked = true`), seluruh modifikasi atau penambahan nilai ditolak dengan HTTP `422 Unprocessable Entity` dan kode error `SCORES_LOCKED`.
  - **Formula Nilai Akhir & Pemeringkatan Dinamis**:
    - Nilai akhir dihitung secara transaksional menggunakan rumus rata-rata tertimbang:
      $$\text{FinalScore} = \sum_{i=1}^n \left( \text{score}_i \times \frac{\text{weight}_i}{100} \right)$$
    - Pemeringkatan dilakukan secara dinamis dan transaksional pada tabel `score_summaries` menggunakan PostgreSQL window function `ROW_NUMBER() OVER (PARTITION BY period_id, path_id, study_program_code ORDER BY final_score DESC, computed_at ASC)` sehingga urutan ranking selalu konsisten dan terkini.
  - **Isolasi Role Program Studi**:
    - Pengguna dengan role `prodi` dibatasi hanya dapat meninjau rekapitulasi nilai dan pemeringkatan pada program studi yang sesuai dengan `scope_id` pada tokennya. Upaya mengakses rekap prodi lain ditolak dengan HTTP `403 Forbidden`.
  - **Ekspor Rekapitulasi Berkas Spreadsheet Excel (.xlsx)**:
    - Endpoint `GET /api/v1/scoring/program/recap/export` memproduksi berkas spreadsheet `.xlsx` resmi menggunakan library `github.com/xuri/excelize/v2` yang memuat header navy blue, border tabel rapi, format angka desimal seragam, rincian komponen nilai, nilai akhir tertimbang, dan nomor urut peringkat.
  - **Inter-Service Consumption untuk Selection Service**:
    - Endpoint internal `GET /internal/summaries?period_id=&program=` dan `POST /internal/locks` disediakan khusus untuk dikonsumsi oleh Selection Service pada tahap perumusan rekomendasi kelulusan.
- **Rasional**: Menjamin keakuratan matematis dan transparansi penilaian seleksi penerimaan mahasiswa baru, melindungi integritas skor dari perubahan tanpa izin setelah masa seleksi berakhir, mengamankan hak akses antar dosen penguji dan program studi, serta mempermudah sidang pleno kelulusan panitia PMB melalui berkas rekapitulasi Excel berstandar tinggi.

---

## 25. Fase 2: Selection Service — Validasi Kuota Mutlak, Penguncian Nilai Transaksional, Persetujuan Pimpinan Dua Arah (Approve/Return), dan Efek Samping Publikasi Pengumuman Terintegrasi (F2-M9)
- **Status**: Diputuskan pada Tahap 8, 24 Sep 2026
- **Keputusan**:
  - **Arsitektur Selection Service Mandiri**:
    - Layanan Selection Service (`pmb-selection-service`) beroperasi sebagai microservice independen pada port `8090` dengan basis data terisolasi `pmb_selection`.
    - Mengikuti standar keamanan ketat `docs/AGENT_RULES.md`: verifikasi signature HMAC API Gateway (`pkg/gatewayauth`), autentikasi token internal (`pkg/internalauth`), migrasi skema SQL versi terstruktur, dan format respons JSON terstandarisasi.
  - **Validasi Kuota Program Studi Mutlak (QUOTA_EXCEEDED)**:
    - Selama perumusan usulan rekomendasi oleh program studi (`POST/PUT /api/v1/selection/program/recommendations`), kuota kapasitas prodi divalidasi langsung dari Admission Service (`GET /internal/quotas`).
    - Jika jumlah pendaftar yang ditetapkan berstatus `pass` melebihi kuota kapasitas yang tersedia, sistem secara mutlak menolak penyimpanan maupun pengajuan dengan status HTTP `422 Unprocessable Entity` dan kode error `QUOTA_EXCEEDED`.
  - **Siklus Penguncian Nilai Transaksional (SCORES_LOCKED)**:
    - Saat program studi mengajukan usulan kelulusan (`POST /api/v1/selection/program/recommendations/:id/submit`), Selection Service memanggil Scoring Service (`POST /internal/locks` dengan `is_locked = true`). Sejak titik ini, dosen penguji tidak lagi diizinkan mengubah skor komponen peserta (`SCORES_LOCKED`).
    - Jika pimpinan mengembalikan berkas usulan untuk ditinjau ulang (`return`), kunci nilai di Scoring Service dibuka kembali (`is_locked = false`) sehingga pihak program studi atau penguji dapat melakukan revisi nilai maupun keputusan kelulusan peserta.
  - **Alur Persetujuan Pimpinan Dua Arah**:
    - Pimpinan universitas memiliki endpoint khusus `GET /api/v1/selection/approval/recommendations` untuk meninjau seluruh usulan prodi yang berstatus `submitted`.
    - Pimpinan dapat menyetujui rekomendasi (`POST /.../approve`) sehingga status berubah menjadi `approved`, atau mengembalikan rekomendasi (`POST /.../return`) disertai catatan telaah tertulis (`note`).
  - **Publikasi Pengumuman Terpusat oleh Admin PMB**:
    - Admin PMB mempublikasikan hasil resmi melalui `POST /api/v1/selection/admin/announcements`.
    - Sistem memastikan seluruh usulan rekomendasi pada periode/gelombang terkait telah disetujui pimpinan sebelum pengumuman dapat diterbitkan.
    - Pada saat pengumuman diterbitkan:
      1. Seluruh penetapan peserta disimpan secara batch ke tabel `results`.
      2. Status pendaftaran di Admission Service diperbarui via internal API menjadi `passed` (lulus) atau `failed` (tidak lulus).
      3. Finance Service dipanggil melalui internal API (`POST /internal/invoices`) untuk membuat invoice tagihan daftar ulang (`reregistration`) bagi setiap calon mahasiswa yang dinyatakan lulus.
      4. Notifikasi kelulusan resmi dikirimkan kepada calon mahasiswa melalui Notification Service (`POST /internal/notifications`).
  - **Perlindungan Akses Hasil Calon Mahasiswa (RESULT_NOT_PUBLISHED)**:
    - Endpoint calon mahasiswa `GET /api/v1/selection/me/result` memproteksi kerahasiaan hasil seleksi. Jika pengumuman belum dirilis resmi oleh Admin PMB, setiap permintaan langsung ditolak dengan HTTP `422 Unprocessable Entity` dan kode error `RESULT_NOT_PUBLISHED`.
    - Setelah rilis resmi, calon mahasiswa dapat melihat keputusan kelulusan, program studi yang menerima, serta rincian invoice tagihan daftar ulang yang telah terbentuk secara otomatis.
- **Rasional**: Menjamin integritas keputusan penerimaan mahasiswa baru agar tidak melebihi daya tampung institusi, menciptakan alur pengawasan berjenjang yang akuntabel antara program studi dan pimpinan universitas, menjaga kerahasiaan hasil sebelum waktu rilis resmi, serta mengotomatisasi side-effects lintas layanan (akademik, keuangan, dan notifikasi) tanpa intervensi manual yang rentan kesalahan.

---

## 26. Fase 2: Finance Bagian 2 — Siklus Cicilan, Presisi Integer Bebas Selisih Pembulatan, Proteksi Biaya Pendaftaran, Pengingat H-3/H-1, dan Laporan Keuangan Multi-Role (F2-M10)
- **Status**: Diputuskan pada Tahap 9, 24 Sep 2026
- **Keputusan**:
  - **Larangan Mutlak Cicilan Biaya Pendaftaran (INSTALLMENT_NOT_ALLOWED)**:
    - Sesuai regulasi institusi PMB dan PRD Bagian 4.5, tagihan biaya pendaftaran (`fee_type == "registration"`) wajib diselesaikan secara penuh (lunas) agar proses seleksi dapat dilanjutkan.
    - Upaya pengajuan cicilan untuk tagihan pendaftaran ditolak seketika oleh sistem dengan status HTTP `422 Unprocessable Entity` dan kode error `INSTALLMENT_NOT_ALLOWED`.
    - Pengajuan cicilan hanya diizinkan untuk tagihan daftar ulang (`reregistration`) yang belum lunas dan belum memiliki pengajuan cicilan aktif lain.
  - **Algoritma Pembagian Cicilan Presisi Integer Rupiah Bebas Selisih**:
    - Nominal uang seluruhnya dikelola dalam bilangan bulat Rupiah (`int64`).
    - Untuk menghindari selisih pembulatan (rounding error) pada nominal ganjil (contohnya Rp 3.333.334 dibagi 3 termin), sistem menghitung nominal dasar `base = totalAmount / numTerms` dan sisa modulus `remainder = totalAmount % numTerms`.
    - Nilai `remainder` ditambahkan sepenuhnya pada termin pertama (`term_1 = base + remainder`, `term_i = base`).
    - Hasil perhitungan: Termin 1 = Rp 1.111.112, Termin 2 = Rp 1.111.111, Termin 3 = Rp 1.111.111. Akumulasi total termin dijamin tepat 100% identik dengan nominal tagihan (Rp 3.333.334), tanpa selisih Rp 1 pun.
  - **Aturan Pembagian Termin dan Tanggal Jatuh Tempo**:
    - Default batas maksimal termin adalah 4 termin, dengan termin pertama minimal 25% dari total tagihan.
    - Tanggal jatuh tempo termin digenerate secara bertahap dengan jeda 30 hari kalender:
      $$\text{DueDate}_i = \text{ApprovalTime} + (30 \times i \text{ hari})$$
  - **Transisi Status Tagihan Berdasarkan Pembayaran Termin**:
    - Saat calon mahasiswa melunasi termin pertama (`term_number == 1`), status invoice bertransisi menjadi `partially_paid`.
    - Status `partially_paid` ini menjadi prasyarat sah bagi calon mahasiswa untuk melanjutkan proses daftar ulang di Student Service (Tahap 10).
    - Status invoice hanya akan berubah menjadi `paid` (lunas total) setelah seluruh termin cicilan diselesaikan.
  - **Pekerjaan Terjadwal & Pengingat Jatuh Tempo (H-3, H-1, Expired)**:
    - Endpoint pemrosesan jatuh tempo `POST /admin/cron/process-due-dates` memeriksa seluruh termin dan tagihan yang belum dibayar.
    - Tagihan/termin yang jatuh tempo dalam 3 hari ke depan dikirimkan notifikasi email pengingat (`payment_reminder_d3`).
    - Tagihan/termin yang jatuh tempo esok hari dikirimkan notifikasi email urgensi (`payment_reminder_d1`).
    - Tagihan/termin yang telah melampaui batas waktu pembayaran (`due_date < now`) secara otomatis ditandai berstatus `expired`.
  - **Laporan Ringkasan Keuangan & Ekspor Spreadsheet Excel Multi-Role**:
    - Endpoint `GET /admin/reports/summary` menyajikan indikator kinerja keuangan agregat (total ditagihkan, total penerimaan terkumpul, piutang tertunggak, serta distribusi status tagihan dan breakdown jenis biaya).
    - Endpoint `GET /admin/reports/export?format=xlsx` menghasilkan berkas spreadsheet Excel `.xlsx` resmi menggunakan library `excelize/v2` yang terdiri dari Sheet 1 (Ringkasan Keuangan Berformat Eksekutif) dan Sheet 2 (Daftar Rincian Seluruh Tagihan & Cicilan).
    - Kedua endpoint laporan dapat diakses oleh role `keuangan`, `pimpinan`, dan `superadmin`. Konfigurasi API Gateway (`routes.yaml`) diperbarui agar prefix `/api/v1/finance/admin` secara sah meneruskan permintaan dari role `pimpinan`.
- **Rasional**: Mencegah kebocoran pendapatan kampus dari selisih pembulatan matematis, menjamin kelancaran arus kas pendaftaran awal, memberikan fleksibilitas pembiayaan bagi mahasiswa berpotensi lewat cicilan daftar ulang yang tertib administrasi, serta membekali pimpinan universitas dengan visibilitas data keuangan real-time untuk pengambilan keputusan strategis.

---

## 27. Fase 2: Student Service — Siklus Pendaftaran Ulang Mahasiswa Baru, Validasi Kliring Tagihan (Lunas / Cicilan Termin 1), Persiapan Outbox Penerbitan NIM, dan Isolasi Program Studi (F2-M11 Bagian Student)
- **Status**: Diputuskan pada Tahap 10, 24 Sep 2026
- **Keputusan**:
  - **Arsitektur Student Service Mandiri**:
    - Layanan Student Service (`pmb-student-service`) beroperasi sebagai microservice independen pada port `8091` dengan basis data terisolasi `pmb_student`.
    - Mengikuti standar keamanan ketat `docs/AGENT_RULES.md`: verifikasi HMAC API Gateway (`pkg/gatewayauth`), autentikasi token internal (`pkg/internalauth`), migrasi skema SQL versi terstruktur, dan format amplop respons JSON standar.
  - **Validasi Mutlak Kelulusan Seleksi (NOT_PASSED)**:
    - Calon mahasiswa hanya dapat mengakses formulir dan menyelesaikan pendaftaran ulang jika pendaftarannya telah dinyatakan **LULUS** (`passed`) oleh panitia seleksi PMB.
    - Calon mahasiswa yang tidak lulus atau berkasnya belum terverifikasi ditolak seketika dengan status HTTP `422 Unprocessable Entity` dan kode error `NOT_PASSED`.
  - **Kliring Pembayaran Tagihan Daftar Ulang (PAYMENT_REQUIRED)**:
    - Sebelum menyelesaikan pendaftaran ulang (`POST /me/reregistration/complete`), sistem memvalidasi status tagihan daftar ulang ke Finance Service.
    - Pembayaran dinyatakan kliring dan sah apabila tagihan berstatus **`paid`** (lunas penuh) **ATAU** berstatus **`partially_paid`** (termin 1 cicilan telah lunas dibayar).
    - Apabila tagihan belum dibayar (`unpaid`), proses pendaftaran ulang ditolak dengan status HTTP `422 Unprocessable Entity` dan kode error `PAYMENT_REQUIRED`.
  - **Atribut Registrasi Mahasiswa Baru Wajib Lengkap**:
    - Calon mahasiswa wajib melengkapi ukuran jas almamater (`S`, `M`, `L`, `XL`, `XXL`, `XXXL`), nama dan nomor telepon kontak darurat, serta menyetujui pernyataan tata tertib akademik kampus (`statement_agreed = true`).
  - **Pola Transaksional & Persiapan Outbox NIM**:
    - Saat pendaftaran ulang diselesaikan (`complete`):
      1. Status pendaftaran ulang diubah menjadi `completed` dengan pencatatan waktu `completed_at`.
      2. Entitas mahasiswa baru dibuat pada tabel `students` dengan status awal `reregistered`.
      3. Pesan kejadian antarlayanan `nim_issuance_requested` disimpan secara transaksional ke tabel `outbox_events` agar siap dikonsumsi oleh NIM Service di Tahap 11.
      4. Status pendaftaran di Admission Service dimutasi menjadi `reregistered` via internal HTTP callback.
  - **Isolasi Role Program Studi & Ekspor Spreadsheet Excel (.xlsx)**:
    - Pengguna dengan role `prodi` dibatasi secara ketat hanya dapat melihat dan mengekspor data mahasiswa baru pada program studi yang menjadi kewenangannya (`scope_id`). Upaya mengakses prodi lain ditolak dengan HTTP `403 Forbidden`.
    - Endpoint ekspor `GET /admin/students/export?format=xlsx` memproduksi berkas spreadsheet resmi Excel menggunakan library `excelize/v2` yang memuat header navy blue, border tabel rapi, serta rincian lengkap pendaftar.
- **Rasional**: Menjamin integritas administratif pendaftaran mahasiswa baru, memastikan kepatuhan kewajiban finansial sebelum registrasi akademik dibuka, memfasilitasi integrasi asinkron penerbitan NIM yang andal dan idempoten, serta mengamankan hak akses data mahasiswa baru antar program studi.

---

## 28. Fase 2: NIM Service — Pencegahan Race Condition dengan SELECT FOR UPDATE, Idempotensi Penerbitan, Pengiriman Otomatis via Transactional Outbox Worker, dan Integrasi Lintas-Layanan (F2-M11 Bagian NIM)
- **Status**: Diputuskan pada Tahap 11, 24 Sep 2026
- **Keputusan**:
  - **Arsitektur NIM Service Mandiri**:
    - Layanan NIM Service (`pmb-nim-service`) beroperasi pada port `8092` dengan basis data terisolasi `pmb_nim`.
    - Mengikuti standar keamanan ketat `docs/AGENT_RULES.md`: verifikasi HMAC API Gateway (`pkg/gatewayauth`), autentikasi token internal (`pkg/internalauth`), migrasi skema SQL terstruktur, dan format amplop respons JSON standar.
  - **Pola Konfigurasi Format NIM Dinamis**:
    - Template pola format default adalah `{YY}{nim_code_prodi}{jalur_digit}{urut:4}`, di mana `{YY}` adalah 2 digit tahun pendaftaran (2026 -> `26`), `{nim_code_prodi}` adalah kode prodi (mis. TI -> `55`, SI -> `57`), `{jalur_digit}` adalah digit jalur masuk (1=reguler, 2=prestasi, 3=beasiswa, 4=pindahan), dan `{urut:4}` adalah nomor urut zero-padded 4 digit.
    - Superadmin dapat mengelola template pola format global maupun spesifik program studi melalui endpoint CRUD `/api/v1/nim/admin/formats`.
  - **Pencegahan Race Condition & Jaminan Unik NIM (SELECT ... FOR UPDATE)**:
    - Sequence counter per tahun dan program studi dikelola pada tabel `nim_sequences`.
    - Pengambilan nomor urut dilakukan secara atomik di dalam database transaction dengan klausa pesimistik `SELECT ... FOR UPDATE`. Mekanisme ini menjamin tidak ada nomor urut/NIM yang terbit ganda (*zero duplicates*) walau 50 atau lebih permintaan masuk secara simultan/konkuren.
  - **Idempotensi Penerbitan per Application ID**:
    - Tabel `nim_issuances` memiliki indeks unik pada `application_id`. Pemanggilan berulang pada endpoint `POST /internal/nim/issue` untuk pendaftaran yang sama akan mengembalikan rekor NIM yang telah diterbitkan sebelumnya tanpa menambah counter sequence.
  - **Publisher Worker Pola Transactional Outbox di Student Service**:
    - Student Service mengimplementasikan background worker (goroutine ticker periodik) dan pemrosesan instan pada saat daftar ulang selesai untuk memproses tabel `outbox_events` (`nim_issuance_requested`).
    - Event dikirimkan secara otomatis ke NIM Service, dan rekor outbox ditandai `processed` dengan timestamp `processed_at`.
  - **Orkestrasi Pasca Penerbitan NIM**:
    - Setelah NIM sah di-commit di basis data NIM, sistem secara otomatis mengeksekusi integrasi ke seluruh layanan terkait:
      1. Memperbarui kolom `nim` dan mengubah status menjadi `active` pada Student Service (`PUT /internal/students/by-application/:id/nim`).
      2. Menambahkan role `mahasiswa` ke akun user di Auth Service (`POST /internal/users/:id/roles`) dan mencabut sesi lama sehingga login baru calon mahasiswa langsung memuat peran mahasiswa aktif.
      3. Memperbarui status pendaftaran di Admission Service menjadi `enrolled` (`PUT /internal/applications/:id/status`).
      4. Mengirimkan notifikasi email/in-app penerbitan NIM ke Notification Service (`POST /internal/notifications`).
- **Rasional**: Mencegah duplikasi nomor identitas akademik yang fatal bagi administrasi perguruan tinggi, menjamin ketersediaan sistem di bawah beban konkurensi tinggi, mengotomatiskan transisi peran calon mahasiswa menjadi mahasiswa aktif, dan memastikan konsistensi status di seluruh mikroservis tanpa ketergantungan transaksi terdistribusi yang rapuh.

---

## 29. Fase 2: Integrasi Akhir — Dashboard Statistik Pimpinan Eksekutif (UC-31), Alur Penuh End-to-End Tanpa Celah, dan Keterlacakan Use Case Lengkap (F2-M12)
- **Status**: Diputuskan pada Tahap 11, 24 Sep 2026
- **Keputusan**:
  - **Dashboard Statistik Pimpinan Eksekutif (UC-31)**:
    - Hak akses peran `pimpinan` (Rektorat/Dekanat) diaktifkan secara terpadu pada lapisan API Gateway (`services/api-gateway/config/routes.yaml`) dan layanan downstream Admission Service (`GET /api/v1/admission/admin/stats`), Finance Service (`GET /api/v1/finance/admin/reports/summary`), Scoring Service (`GET /api/v1/scoring/program/recap`), Selection Service (`GET /api/v1/selection/approval/recommendations`), Student Service (`GET /api/v1/student/admin/students`), dan NIM Service (`GET /api/v1/nim/admin/issuances`).
    - Pimpinan universitas mendapatkan agregasi metrik pendaftaran (total aplikasi, sebaran status hingga `enrolled`, sebaran per jalur & prodi), kesehatan finansial penerimaan kas (total tagihan terbit, pembayaran lunas, ringkasan cicilan aktif), ranking hasil seleksi calon mahasiswa, serta monitoring registrasi ulang mahasiswa baru dan penomoran NIM.
    - Hak akses dilindungi secara ketat: role non-pimpinan/non-admin (seperti `calon_mahasiswa` atau `dosen`) yang mencoba mengakses endpoint eksekutif ini ditolak seketika dengan status HTTP `403 Forbidden`.
  - **Verifikasi Alur Menyeluruh (Full Flow End-to-End Suite)**:
    - Skrip pengujian `scripts/e2e.sh` mencakup 118 skenario komprehensif yang menguji alur PMB dari registrasi akun awal, verifikasi OTP, pemilihan program studi & jalur, pembuatan dan pelunasan invoice tagihan pendaftaran, pengisian biodata terenkripsi AES-256-GCM & OCR KTP, pengunggahan dokumen persyaratan dengan validasi magic bytes, verifikasi berkas & revisi dokumen panitia, penjadwalan ujian & kartu peserta PDF dengan QR code, input nilai oleh dosen & kalkulasi rata-rata tertimbang, rekomendasi prodi dengan pembatasan kuota mutlak, persetujuan pimpinan (approve/return), pengumuman kelulusan resmi, penerbitan tagihan daftar ulang dan pengajuan termin cicilan presisi integer, pelunasan termin 1, formulir daftar ulang, penerbitan NIM dengan penguncian pesimistik `SELECT ... FOR UPDATE` bebas race condition, pembaruan role `mahasiswa` otomatis, hingga dashboard eksekutif pimpinan.
  - **Penyelesaian Seluruh Use Case (UC-01 s/d UC-33)**:
    - Seluruh use case pada tabel keterlacakan (PRD-fase-2.md Bagian 3) dinyatakan **Selesai**, termasuk ekspor laporan spreadsheet Excel .xlsx (UC-26), dashboard statistik pimpinan (UC-31), dan pengelolaan master data (UC-32).
- **Rasional**: Memberikan visibilitas terpadu kepada para pengambil keputusan universitas tanpa melanggar prinsip desentralisasi basis data mikroservis, menjamin seluruh alur operasional PMB bebas hambatan dari hulu ke hilir, serta memverifikasi kesiapan produksi sistem backend PMB sesuai seluruh tolok ukur Definition of Done.

---

## 30. Keputusan Default CR-006, CR-007, dan CR-008 (P1–P4, M1–M6, S1)
- **Status**: default, menunggu konfirmasi lead (29 September 2026, sumber: `PRD-TAMBAHAN-CR006-008.md` Bagian 5)
- **Rangkuman Keputusan**:

| Kode | Keputusan | Status Konfirmasi |
|---|---|---|
| **P1** | Pindah jalur oleh calon mahasiswa setelah bayar hanya lewat permintaan yang disetujui admin, maksimal 1 kali, dan hanya sebelum dijadwalkan ujian | default, menunggu konfirmasi lead |
| **P2** | Selisih biaya lebih mahal ditagihkan (invoice ADJ); lebih murah tidak dikembalikan otomatis (dicatat untuk ditinjau keuangan) | default, menunggu konfirmasi lead |
| **P3** | Setelah pindah jalur, dokumen diverifikasi ulang bila jalur baru punya persyaratan berbeda | default, menunggu konfirmasi lead |
| **P4** | Nomor pendaftaran tetap saat pindah jalur | default, menunggu konfirmasi lead |
| **M1** | Pengunduran diri sebelum lulus langsung berlaku; setelah lulus/daftar ulang/NIM perlu persetujuan admin | default, menunggu konfirmasi lead |
| **M2** | Pengunduran diri wajib konfirmasi kata sandi dan alasan berkategori | default, menunggu konfirmasi lead |
| **M3** | Uang yang sudah dibayar tidak dikembalikan otomatis; ditandai untuk ditinjau keuangan (`refund_review`) | default, menunggu konfirmasi lead |
| **M4** | NIM yang sudah terbit dibatalkan (`voided`), tidak dipakai ulang | default, menunggu konfirmasi lead |
| **M5** | Setelah mundur, boleh mendaftar lagi di gelombang yang masih buka | default, menunggu konfirmasi lead |
| **M6** | Kursi yang kosong karena mundur dilaporkan ke prodi; promosi cadangan (waitlist) tetap manual | default, menunggu konfirmasi lead |
| **S1** | Uji otomatis tetap memakai mock (`PAYMENT_PROVIDER=mock`); uji Duitku sandbox manual lewat script PowerShell | default, menunggu konfirmasi lead |

- **Rasional & Implikasi Arsitektural**:
  - **P1 & P4**: Menjaga stabilitas nomor pendaftaran (`registration_number`) dan mencegah abuse pergantian jalur berulang kali setelah proses verifikasi berkas atau penjadwalan ujian dimulai.
  - **P2**: Mempertahankan integritas finansial: biaya tambahan yang timbul diterbitkan sebagai tagihan penyesuaian (`INV-REG-{YYYY}-{urut}-ADJ`), dan tidak ada mekanisme auto-refund untuk mencegah kerugian operasional dan manipulasi transaksi.
  - **P3**: Memastikan kepatuhan akademik bahwa berkas yang belum pernah dipenuhi pada jalur sebelumnya wajib diunggah kembali (status kembali ke `form_open` jika ada berkas belum lengkap).
  - **M1 & M2**: Mencegah pengunduran diri palsu atau tidak sengaja dengan otentikasi kata sandi langsung ke Auth Service (`POST /internal/users/:id/verify-password`) dan pembatasan wewenang berjenjang untuk mahasiswa yang sudah melewati fase seleksi akhir.
  - **M3 & M4**: NIM adalah identitas universitas permanen yang tidak boleh didaur ulang (*no sequence reuse*); status `voided` mengamankan audit trail. Pengembalian dana dilakukan manual oleh pihak keuangan.
  - **M5 & M6**: Memberikan kesempatan kedua bagi pendaftar tanpa merusak alur seleksi normal; pelepasan kuota memberi transparansi kursi kosong bagi panitia program studi.
  - **S1**: Memastikan siklus CI/CD dan pengujian regression `scripts/e2e.sh` tetap deterministik, cepat, dan independen dari ketergantungan koneksi pihak ketiga/jaringan eksternal.

---

## 31. Keputusan Default CR-009 dan CR-010 (K11–K18, Q1–Q5)
- **Status**: default, menunggu konfirmasi lead (29 September 2026, sumber: `PRD-TAMBAHAN-CR009-010.md` Bagian 4)
- **Rangkuman Keputusan**:

| Kode | Keputusan | Status Konfirmasi |
|---|---|---|
| **K11** | CBT terbuka setelah formulir lunas + biodata lengkap; verifikasi berkas paralel; kelulusan hanya bisa dipublikasikan bila berkas terverifikasi | default, menunggu konfirmasi lead |
| **K12** | NIM diterbitkan admin (satuan/batch), bukan otomatis (`NIM_ISSUANCE_MODE=manual`) | default, menunggu konfirmasi lead |
| **K13** | Konfirmasi lunas manual hanya keuangan/superadmin dengan bukti & nomor referensi mutasi unik | default, menunggu konfirmasi lead |
| **K14** | Ambang kelulusan global per gelombang (default 70.0); di bawah ambang → "Evaluasi Pleno", bukan otomatis tidak lulus | default, menunggu konfirmasi lead |
| **K15** | Batas bayar formulir 3 hari kerja, bisa diperpanjang maks. 2 kali | default, menunggu konfirmasi lead |
| **K16** | Role baru `dekan` (scope fakultas) untuk tab Dekanat di Dashboard Institusi | default, menunggu konfirmasi lead |
| **K17** | Biaya transaksi pembayaran ditanggung kampus (tampil Rp 0 ke pendaftar) | default, menunggu konfirmasi lead |
| **K18** | Integrasi SIAKAD lewat ekspor file dulu | default, menunggu konfirmasi lead |

- **Pertanyaan Terbuka & Keputusan Default Sementara**:

| No | Pertanyaan | Default yang Diterapkan |
|---|---|---|
| **Q1** | Desain kuitansi: "pilihan terkunci dan tidak dapat diubah lagi", sedangkan CR-006 membolehkan pindah jalur setelah bayar lewat persetujuan. Mana yang dipakai? | CR-006 tetap berlaku; teks kuitansi dikonfigurasi dinamis ("Pembayaran lunas. Pilihan jalur dan program studi terkunci; perubahan hanya melalui pengajuan ke panitia.") |
| **Q2** | Format NIM berbeda antar halaman: halaman 18 `[YY][Kode Prodi 2][Urut 4]`, halaman 24 `{YY}{NIM}{JALUR}{URUT}` | Template dapat dikonfigurasi; default tetap `{YY}{kode NIM prodi}{kode jalur}{urut 4}` (sesuai PRD & halaman 24) |
| **Q3** | Nomor VA di desain berpola tetap (mis. `988 2026 01 94461`, turunan nomor registrasi) — apakah butuh VA tetap per pendaftar atau VA dinamis Duitku sudah cukup? | VA dinamis Duitku; nomor di desain dianggap contoh ilustratif |
| **Q4** | Penyedia WhatsApp Gateway resmi untuk broadcast? | Adapter `log` default; driver penyedia resmi ditambahkan setelah dipilih vendor institusi |
| **Q5** | Pengaman konfirmasi lunas manual: perlu persetujuan dua orang keuangan? | Tidak (`MANUAL_CONFIRM_SECOND_APPROVAL=false` default) |

- **Rasional & Implikasi Arsitektural**:
  - **K11**: Mengakomodasi kebutuhan pendaftar untuk segera menyelesaikan ujian CBT daring tanpa terhambat antrean verifikasi berkas ijazah/SKL, namun gerbang kelulusan akhir tetap terkunci sampai berkas dinyatakan sah.
  - **K12**: Menghindari penerbitan NIM liar secara otomatis pada skenario di mana panitia akademik perlu memverifikasi kelengkapan berkas fisik sebelum alokasi nomor induk mahasiswa resmi.
  - **K13**: Menutup celah kecurangan audit finansial: setiap pencatatan pelunasan manual wajib menyertakan identitas referensi transaksi mutasi perbankan yang unik (menolak rekor ganda) dan file bukti bayar.
  - **K14**: Memberikan fleksibilitas pada rapat pleno pimpinan untuk meninjau calon mahasiswa potensial (kategori `plenary`) sebelum vonis kelulusan final.
  - **K15**: Mencegah penumpukan invoice pendaftaran basi dan mempermudah perpanjangan jatuh tempo tanpa harus membatalkan pendaftaran.
  - **K16**: Mengimplementasikan prinsip hak akses terkecil (*least privilege*) berjenjang: Dekan dapat memonitor seluruh prodi di bawah naungan fakultasnya tanpa wewenang lintas fakultas lain.
  - **K17**: Memberikan kepastian nominal pembayaran yang bersih di sisi calon mahasiswa (*zero fee deduction* di UI).
  - **K18**: Mengisolasi kompleksitas API eksternal SIAKAD pada fase ini, memprioritaskan format ekspor standar xlsx/csv.









