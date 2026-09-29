# Change Requests Log (CHANGE_REQUESTS.md)

Dokumen ini mencatat seluruh permintaan perubahan (*Change Request* / CR) yang diajukan oleh lead (Pak Kemal) atau pemangku kepentingan selama pengembangan Sistem PMB Fase 2. Setiap perubahan dianalisis dampaknya terhadap arsitektur, basis data, endpoint, pengujian, dan tahapan timeline sebelum diimplementasikan.

---

## Tabel Log Change Request

| No. CR | Tanggal Pengajuan | Ringkasan Perubahan | Sumber / Pengaju | Service / Komponen Terdampak | Tahap Terdampak | Status | Dokumen Terkait |
|---|---|---|---|---|---|---|---|
| **CR-001** | 2026-09-24 | Integrasi Payment Gateway Duitku (Duitku POP / `createInvoice`) menggantikan mock-only; MockProvider tetap dipertahankan untuk pengujian otomatis | Pak Kemal Ramadhan (Lead / Product Owner) | Finance Service (`services/finance-service`), API Gateway (`services/api-gateway`), Compose (`deploy/docker-compose.yml`), Scripts (`scripts/e2e.sh`) | Tahap 4 (Finance 1) & Tahap 9 (Finance 2) | **done** | `PRD-LENGKAP.md` Bagian D, `docs/DECISIONS.md` |
| **CR-002** | 2026-09-25 | Batalkan pendaftaran (sebelum bayar) & daftar ulang setelah tidak lulus, riwayat pendaftaran, eligibility | Pak Kemal Ramadhan (Lead / Product Owner) | Admission Service, Finance Service, Notification Service, Student/NIM | Tahap 3 (Admission 1), Tahap 4 (Finance 1), Tahap 5, Tahap 8/10/11 | **done** | `PRD-LENGKAP.md` Bagian D.5–D.8 |
| **CR-003** | 2026-09-25 | CBT (Computer Based Test): bank soal, ujian online, timer server, autosave, penilaian otomatis & esai dinilai dosen | Pak Kemal Ramadhan (Lead / Product Owner) | Exam Service, Scoring Service, Gateway, Compose | Tahap 6 (Exam Service) & Tahap 7 (Scoring Service) | **done** | `PRD-LENGKAP.md` Bagian D.9–D.12 |
| **CR-004** | 2026-09-25 | Penyesuaian backend dengan desain frontend Figma (detail pembayaran, stepper progress pendaftaran, polling status) | Pak Kemal Ramadhan (Lead / Product Owner) | Admission Service, Finance Service, Gateway, Docs | Tahap 3, Tahap 4, Docs | **done** | `PRD-LENGKAP.md` Bagian D.13, `docs/FE-BE-GAP.md` |
| **CR-005** | 2026-09-26 | View endpoint dashboard & keselarasan tampilan frontend (stepper progress, status real-time, aggregator dashboard) | Pak Kemal Ramadhan (Lead / Product Owner) | Admission Service, Finance Service, Gateway | Tahap 12 | **done** | `PRD-LENGKAP.md` Bagian D.14–D.18 |
| **CR-006** | 2026-09-29 | Riwayat pendaftaran & pindah jalur (langsung / via permintaan admin), immutable DB triggers, reprice/ADJ invoice, penghitungan ulang dokumen | Pak Kemal Ramadhan (Lead / Product Owner) | Admission Service, Finance Service, Exam Service, API Gateway | Tahap 13 / CR-006 | **done** | `PRD-TAMBAHAN-CR006-008.md` Bagian 1 |
| **CR-007** | 2026-09-29 | Fitur pengunduran diri resmi (verifikasi password Auth, pembatalan tagihan & refund review Finance, pelepasan kuota Selection, pencabutan role Auth, NIM voided, bukti PDF) | Pak Kemal Ramadhan (Lead / Product Owner) | Admission Service, Auth Service, Finance Service, Exam Service, Selection Service, Student Service, NIM Service, API Gateway | Tahap 13 / CR-007 | **done** | `PRD-TAMBAHAN-CR006-008.md` Bagian 2 |
| **CR-008** | 2026-09-29 | Uji integrasi Duitku Sandbox sungguhan (pemisahan POP & WebAPI base URL, script powershell uji sandbox, penanganan PAYMENT_PROVIDER_ERROR, rekonsiliasi) | Pak Kemal Ramadhan (Lead / Product Owner) | Finance Service, API Gateway, Scripts, Docs | Tahap 13 / CR-008 | **done** | `PRD-TAMBAHAN-CR006-008.md` Bagian 3, `docs/DUITKU-SANDBOX.md` |

---

## Catatan Analisis Dampak & Riwayat CR

### CR-001 — Payment Gateway Duitku (24 September 2026)
- **Latar Belakang**: Lead memutuskan menggunakan provider resmi Duitku.
- **Dampak**: Finance Service (`DuitkuProvider` POP `createInvoice`, HMAC-SHA256 signature, webhook parsing, checkStatus), migrations field Duitku, fee schemes CRUD.

### CR-002 — Batalkan Pendaftaran & Daftar Lagi Setelah Tidak Lulus (25 September 2026)
- **Pesan Lead**: "Bikin kondisi tampilannya jika dibatalkan pendaftarannya dia bisa pilih lagi, sama kalau dia dinyatakan tidak lulus dia bisa tambah pendaftaran untuk daftar ke jalur dan pilihan lain. Jadi ada fitur batalkan pendaftaran dengan syarat fitur itu aktif sebelum dia bayar biaya pendaftaran; kalau sudah bayar biaya pendaftaran udah gak bisa dibatalkan."
- **Aturan Bisnis**:
  - Status baru `cancelled` pada state machine pendaftaran. Transisi hanya dari `draft` atau `awaiting_payment`.
  - Sinkron panggil Finance `POST /internal/invoices/:id/cancel`. Hanya batal jika belum ada pembayaran `success` atau `awaiting_verification`.
  - Pembayaran pending Duitku saat dibatalkan ditandai `cancelled`. Jika callback sukses tiba belakangan, tandai `paid_after_cancel`.
  - Satu user hanya boleh punya satu pendaftaran aktif.
  - Setelah `failed`: boleh daftar lagi dengan kombinasi prodi/jalur berbeda (gelombang sama tolak `SAME_CHOICE_ALREADY_FAILED`).
  - Setelah `passed`/`reregistered`/`enrolled`: tolak `ALREADY_ACCEPTED` (satu user satu NIM).
- **Endpoints**:
  - `POST /admission/applications/me/cancel`
  - `GET /admission/applications/me/history`
  - `GET /admission/applications/me/eligibility`
  - `POST /internal/invoices/:id/cancel` di Finance

### CR-003 — CBT (Computer Based Test) (25 September 2026)
- **Pesan Lead**: "Tinggal halaman detail CBT-nya yang belum, itu kan baru awal CBT-nya, ini yang ada soal dan lain-lainnya."
- **Ruang Lingkup**:
  - Ujian online berbasis soal dengan timer server: `min(started_at + duration, session.end_at)`.
  - Tipe soal: `single_choice`, `multiple_choice`, `true_false`, `essay`.
  - Soal dan pilihan diacak per peserta; kunci jawaban aman (tidak bocor ke peserta).
  - Autosave per soal + event logging (focus lost, fullscreen exit).
  - Penilaian otomatis objektif saat submit; esai dinilai dosen; nilai akhir dikirim ke Scoring.
- **Tabel Baru di `pmb_exam`**: `question_banks`, `questions`, `question_options`, `cbt_exams`, `cbt_exam_rules`, `cbt_attempts`, `cbt_answers`, `cbt_events`.

### CR-004 — Penyesuaian Backend dengan Desain Frontend (Figma) (25 September 2026)
- **Arahan Lead**: Backend menyesuaikan desain frontend di Figma.
- **Ruang Lingkup**:
  - Halaman detail pembayaran: `GET /invoices/me/:id` mengembalikan rincian lengkap, metode bayar, VA/link Duitku, expiry, riwayat pembayaran, kwitansi.
  - Polling status pembayaran: `GET /invoices/me/:id/payments/:payment_id/status`.
  - Stepper langkah pendaftaran: `GET /admission/applications/me/progress` mengembalikan status tiap langkah (`done`, `current`, `locked`).
  - Format respon metadata konsisten di setiap list.

### CR-005 — View Endpoints Dashboard & Agregator Frontend (26 September 2026)
- **Arahan Lead**: Endpoint agregasi tampilan frontend dashboard pendaftar dan sinkronisasi status UI.
- **Ruang Lingkup**:
  - Endpoint ringkasan progres pendaftaran calon mahasiswa (`GET /applications/me/progress` & `GET /applications/me/dashboard`).
  - Keterpaduan status tahapan dan data pendukung navigasi UI.
- **Status**: **done** (Tahap 12).

### CR-006 — Riwayat & Pindah Jalur (29 September 2026)
- **Arahan Lead**: Calon mahasiswa dapat pindah jalur dan/atau prodi pilihan dengan aturan sesuai tahap pendaftarannya, tercatat sebagai riwayat immutable (tidak bisa diubah/dihapus, trigger DB menolak UPDATE/DELETE).
- **Aturan Bisnis**:
  - `draft`, `awaiting_payment`: pindah langsung via endpoint pilihan, tagihan disesuaikan (reprice invoice jika biaya berbeda dan belum ada pembayaran sukses/pending).
  - `form_open`, `submitted`, `revision_required`, `verified`: mengajukan permintaan (`pending`) yang harus disetujui `admin_pmb`. Maksimal 1 kali disetujui per pendaftaran (`PATH_CHANGE_MAX_AFTER_PAYMENT=1`).
  - `exam_scheduled`: calon mahasiswa tidak boleh; admin boleh langsung (peserta dikeluarkan dari sesi ujian lama di Exam Service via internal API).
  - Sudah ada nilai / `passed` / `failed` / `reregistered` / `enrolled` / `cancelled` / `withdrawn`: tidak boleh pindah jalur.
  - Selisih biaya: jika lebih mahal -> Finance terbitkan tagihan adjustment `INV-REG-{YYYY}-{urut}-ADJ`; jika lebih murah -> dicatat di riwayat & notifikasi keuangan (tidak ada refund otomatis).
  - Persyaratan dokumen: dihitung ulang untuk jalur baru.
  - Kompatibilitas CR-005: `GET /applications/me/dashboard` memuat field `pending_request`.
- **Status**: **open** (menunggu persetujuan rencana implementasi).

### CR-007 — Pengunduran Diri (29 September 2026)
- **Arahan Lead**: Calon mahasiswa yang sudah membayar dapat mengundurkan diri secara resmi, tercatat, dan berdampak benar ke seluruh sistem (sebelum bayar tetap memakai Batalkan pendaftaran CR-002).
- **Aturan Bisnis**:
  - `draft`, `awaiting_payment`: tolak `WITHDRAWAL_NOT_ALLOWED`, arahkan ke pembatalan CR-002.
  - `form_open`, `submitted`, `revision_required`, `verified`, `exam_scheduled`: langsung berlaku setelah konfirmasi.
  - `passed`, `reregistered`, `enrolled` (NIM terbit, mahasiswa belum aktif): perlu persetujuan `admin_pmb`.
  - Mahasiswa sudah `active`: tolak `WITHDRAWAL_NOT_ALLOWED` (arahkan ke bagian akademik).
  - Konfirmasi wajib: kategori alasan, teks alasan (10–500 karakter), centang pernyataan, dan kata sandi akun (diverifikasi lewat Auth internal `POST /internal/users/:id/verify-password`).
  - Dampak lintas service idempoten (outbox):
    - Admission: status pendaftaran -> `withdrawn`.
    - Finance: tagihan belum dibayar dibatalkan; pembayaran pending dibatalkan (jika callback sukses datang belakangan -> `paid_after_cancel`). Pembayaran yang sudah masuk ditandai `refund_review` untuk penanganan manual bagian keuangan.
    - Exam: peserta dikeluarkan dari sesi; attempt CBT yang sedang berjalan dikumpulkan otomatis.
    - Selection: hasil ditandai `withdrawn`, kuota prodi dilepas, notifikasi `seat_released` ke prodi & admin.
    - Student: pembatalan status registrasi.
    - NIM: nomor yang terbit diberi status `voided` (tidak dipakai ulang).
    - Auth: role `mahasiswa` dicabut via internal API `DELETE /internal/users/:id/roles/:code`.
  - Surat bukti pengunduran diri format PDF dengan QR code verifikasi publik (`GET /admission/withdrawals/verify/:code`).
  - Pengunduran diri oleh admin atas nama pendaftar (unggah surat PDF/JPG <= 5 MB).
- **Status**: **open** (menunggu persetujuan rencana implementasi).

### CR-008 — Uji Integrasi Duitku Sandbox Sungguhan (29 September 2026)
- **Arahan Lead**: Duitku mode sandbox dapat dipakai tanpa verifikasi dokumen. Integrasi Duitku wajib diuji terhadap sandbox sungguhan dengan script PowerShell pengujian otomatis/interaktif, bukan hanya MockProvider.
- **Ruang Lingkup**:
  - Verifikasi endpoint & rumus signature terbaru dari docs.duitku.com, didokumentasikan di `docs/DUITKU-SANDBOX.md`.
  - Pemisahan base URL konfigurasi: `DUITKU_POP_BASE_URL` dan `DUITKU_WEBAPI_BASE_URL`.
  - Pemetaan error Duitku -> `502 PAYMENT_PROVIDER_ERROR` (pesan Bahasa Indonesia; tanpa membocorkan API key di log).
  - Konfigurasi `DUITKU_RECONCILE_INTERVAL` untuk percepatan uji rekonsiliasi.
  - Script pengujian Windows `scripts/duitku-sandbox-check.ps1` (ASCII-only) dengan alur pembuatan pendaftaran, penarikan metode bayar sandbox, instruksi simulasi bayar VA, polling status, verifikasi pelunasan via callback dan rekonsiliasi.
  - Unit test tetap menggunakan mock/httptest (tidak menembak Duitku saat build/CI); `scripts/e2e.sh` tetap `PAYMENT_PROVIDER=mock`.
- **Status**: **open** (menunggu persetujuan rencana implementasi).

