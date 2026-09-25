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
