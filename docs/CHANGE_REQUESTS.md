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
| **CR-009** | 2026-09-29 | Portal Admin (8 modul) & RBAC berbasis permission (tabel permissions/role_permissions, role dekan scope fakultas, header X-User-Permissions di gateway, pkg/rbac, pemetaan permission, 8 modul portal admin) | Desain Figma PMB v2 & Arahan Lead (Pak Kemal) | Seluruh service (Auth, Gateway, Organization, Notification, Admission, Finance, Identity, Exam, Scoring, Selection, Student, NIM) | Tahap 14 / CR-009 | **done** | `PRD-TAMBAHAN-CR009-010.md` Bagian 1 & 2, `docs/RBAC.md` |
| **CR-010** | 2026-09-29 | Penyesuaian desain Portal Calon Mahasiswa v2 (stepper 6 tahap, agenda & ringkasan, options admission_open & biaya formulir, 5 berkas wajib, tab pembayaran VA/Transfer/QRIS + badge + biaya gratis + jatuh tempo 3 hari kerja, pembukaan CBT otomatis K11, receipt-view) | Desain Figma PMB v2 & Arahan Lead (Pak Kemal) | Admission, Finance, Identity, Exam, Selection, Gateway, Organization | Tahap 14 / CR-010 | **done** | `PRD-TAMBAHAN-CR009-010.md` Bagian 3 |
| **CR-011** | 2026-10-04 | Konsolidasi 13 service menjadi 3 service (gateway, pmb-service, ocr-service) & 2 database (pmb_auth, pmb_app) | Arahan Lead / PRD TAMBAHAN CR011 | Seluruh 13 service backend | Fase K (M0–M5) | **done** | `docs/DECISIONS.md` Bagian 32, `docs/CONSOLIDATION-REPORT.md` |
| **CR-012** | 2026-10-04 | Auth 2 Versi (Versi A: SSO Core mandiri, Versi B: Auth PMB) & skema users-service_accounts-user_accounts | Arahan Lead / PRD TAMBAHAN CR011 | Auth Service, Gateway | Fase K (M6–M8) | **done** | `docs/DECISIONS.md` Bagian 32 |
| **CR-013** | 2026-10-04 | Kesiapan Rilis Aplikasi Pendaftaran PMB (audit alur camaba, jalur gratis Rp 0, integrasi Duitku produksi, dokumen rilis) | Arahan Lead / PRD TAMBAHAN CR011 | Admission, Finance, Gateway, Docs | Fase R (R0–R4) | **done** | `docs/RELEASE-CHECKLIST.md`, `docs/DUITKU-PRODUKSI.md` |
| **CR-014** | 2026-10-04 | Modul Referal Guru BK, Pembayaran Kolektif & Komisi (150 mitra guru BK, komisi snapshot, setoran & buku besar) | Arahan Lead / PRD TAMBAHAN CR011 | pmb-service (referral), gateway | Fase F (F1–F5) | **done** | `docs/REFERRAL.md` |
| **CR-015** | 2026-10-13 | Penyesuaian Launching (Direct Payment Duitku v2 tanpa popup, WA Group Onboarding, NIM SK Rektor 065, Ekspor SEVIMA 62-kolom, Identitas Kampus Dinamis, Dashboard Mitra) | Arahan Lead / PRD TAMBAHAN CR011 | Seluruh modul pmb-service & Gateway | Fase G (G1–G6) | **done** | `PRD-TAMBAHAN-CR011.md` Bagian VI |

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
- **Status**: **done** (Tahap 13 / CR-006).

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
- **Status**: **done** (Tahap 13 / CR-007).

### CR-008 — Uji Integrasi Duitku Sandbox Sungguhan (29 September 2026)
- **Arahan Lead**: Duitku mode sandbox dapat dipakai tanpa verifikasi dokumen. Integrasi Duitku wajib diuji terhadap sandbox sungguhan dengan script PowerShell pengujian otomatis/interaktif, bukan hanya MockProvider.
- **Ruang Lingkup**:
  - Verifikasi endpoint & rumus signature terbaru dari docs.duitku.com, didokumentasikan di `docs/DUITKU-SANDBOX.md`.
  - Pemisahan base URL konfigurasi: `DUITKU_POP_BASE_URL` dan `DUITKU_WEBAPI_BASE_URL`.
  - Pemetaan error Duitku -> `502 PAYMENT_PROVIDER_ERROR` (pesan Bahasa Indonesia; tanpa membocorkan API key di log).
  - Konfigurasi `DUITKU_RECONCILE_INTERVAL` untuk percepatan uji rekonsiliasi.
  - Script pengujian Windows `scripts/duitku-sandbox-check.ps1` (ASCII-only) dengan alur pembuatan pendaftaran, penarikan metode bayar sandbox, instruksi simulasi bayar VA, polling status, verifikasi pelunasan via callback dan rekonsiliasi.
  - Unit test tetap menggunakan mock/httptest (tidak menembak Duitku saat build/CI); `scripts/e2e.sh` tetap `PAYMENT_PROVIDER=mock`.
- **Status**: **done** (Tahap 13 / CR-008).

### CR-009 — Portal Admin & RBAC Berbasis Permission (29 September 2026)
- **Arahan Lead & Sumber Desain**: Desain Figma PMB v2 (24 halaman: 13–24 portal admin); arahan lead: "di admin pakai sistem RBAC dan ROLE, yang bisa akses admin bukan hanya admin super tapi keuangan, prodi juga dengan akses terbatas".
- **Ruang Lingkup**:
  - Auth Service: tabel `permissions` dan `role_permissions`, seeder matriks permission 1.4, role `dekan` (scope `faculty`), klaim `perms` pada access token, endpoint `GET /auth/me/permissions`, `GET /auth/admin/permissions`, `GET/PUT /auth/admin/roles/:code/permissions`.
  - API Gateway: header `X-User-Permissions` (comma-separated), pembersihan header palsu dari client, route guard kasar (staf/admin) sementara pengecekan hak aksi dan scope dilakukan di downstream service.
  - Shared package `pkg/rbac`: middleware `RequirePermission(code...)` dan helper scope `ScopeFilter`.
  - 8 Modul Portal Admin:
    1. Ringkasan Eksekutif & KPI (`/admission/admin/overview?period_id=`)
    2. Verifikasi Berkas (`/admission/admin/document-queue`, `/admission/admin/applications/:id/documents`, 5 berkas wajib)
    3. Rekonsiliasi Keuangan & Virtual Account (`/finance/admin/reconciliation`, konfirmasi manual referensi mutasi unik)
    4. Seleksi Akademik & CBT (`/selection/admin/scoreboard`, ambang 70.0, input nilai wawancara, SK Rektor, aturan pilihan kedua)
    5. Penetapan NIM & PKKMB (`/student/admin/enrolment`, penerbitan NIM manual/batch K12, gugus PKKMB, ekspor SIAKAD)
    6. Dashboard Institusional (`/admission/admin/institution-dashboard`, kuota, keketatan, pendaftar per fakultas/prodi)
    7. Pusat Pelaporan, Cetak Resmi, Broadcast & Buku Besar (ekspor csv/xlsx, PDF Berita Acara nomor unik `{urut 3 digit}/BA-PMB/...`, broadcast notifikasi massal, buku besar kas masuk)
    8. Master Data Organisasi: Fakultas & Program Studi (`admission_open` buka/tutup prodi PMB, estimasi UKT).
  - Notification Service: tabel `activity_events`, internal ingestion `POST /internal/activity`, baca log aktivitas `GET /notifications/admin/activity`.
  - Organization Service: profil institusi field tambahan (kop surat berita acara, rekening bank transfer institusi, dsb.).
- **Status**: **done** (Tahap 14 / CR-009).

### CR-010 — Penyesuaian Desain Portal Calon Mahasiswa v2 (29 September 2026)
- **Arahan Lead & Sumber Desain**: Desain Figma PMB v2 (24 halaman: 05–12 portal calon mahasiswa).
- **Ruang Lingkup**:
  - Beranda (05): Stepper 6 tahap (`profile`, `path`, `payment`, `documents`, `selection`, `reregistration`), jadwal & agenda seleksi, summary cards, kontak institusi.
  - Biodata (06): Field `report_average` (rata-rata rapor) di riwayat sekolah.
  - Jalur & Prodi (07): Pilihan prodi hanya `admission_open = true`, biaya formulir per jalur di options, status gabungan.
  - Berkas Persyaratan (08): 5 dokumen wajib (IJAZAH_SKL, KTP_KIA_KP, KK, PAS_FOTO, SURAT_SEHAT), sinkronisasi pas foto profil & berkas.
  - Pembayaran (09): Tab Virtual Account / Transfer Bank / QRIS, badge bank mitra/utama, fee transaksi gratis ditanggung kampus, jatuh tempo 3 hari kerja (`REGISTRATION_PAYMENT_DUE_BUSINESS_DAYS=3`), renew tagihan kedaluwarsa.
  - Pelaksanaan CBT (10): K11 kelayakan CBT (biaya lunas + biodata lengkap langsung membuka modul CBT tanpa menunggu berkas terverifikasi, verifikasi berkas berjalan paralel).
  - Pengumuman & Registrasi (11): Penugasan gugus PKKMB calon mahasiswa baru.
  - Kuitansi Resmi (12): Endpoint `GET /finance/me/invoices/:id/receipt-view`, kuitansi lunas, teks status penguncian pilihan jalur.
- **Status**: **done** (Tahap 14 / CR-010).

### CR-011 — Konsolidasi 3 Service (Modular Monolith) (4 Oktober 2026)
- **Arahan Lead**: Konsolidasi 13 service menjadi 3 service aplikasi (`gateway`, `pmb-service`, `ocr-service`) dan 2 database (`pmb_auth`, `pmb_app`) tanpa mengubah kontrak API dan perilaku endpoint publik.
- **Ruang Lingkup**:
  - `gateway`: penggabungan api-gateway dan auth-service dengan database `pmb_auth`.
  - `pmb-service`: modular monolith yang mencakup modul `admission`, `finance`, `identity`, `organization`, `exam`, `scoring`, `selection`, `student`, `nim`, `notification`, dan `referral`. Schema terpisah per modul di DB `pmb_app`, integrasi antar modul via facade Go (in-process outbox).
  - `ocr-service`: service terpisah (Python), opsional (`OCR_ENABLED=false` mengembalikan `503 OCR_DISABLED`).
- **Status**: **open** (Fase K / M0–M5).

### CR-012 — Auth Dua Versi & Persiapan SSO (4 Oktober 2026)
- **Arahan Lead**: Auth 2 versi: Versi A (Auth Kampus / SSO Core) dan Versi B (Auth PMB), skema `users` (UserTabel) – `service_accounts` (ServiceAccount) – `user_accounts` (UserAccount) + `user_account_roles`.
- **Ruang Lingkup**:
  - Versi A: SSO Core mandiri (`cmd/sso`), OpenID Connect Authorization Code + PKCE, JWKS & rotasi kunci, discovery `.well-known/openid-configuration`, `/oauth2/*`, `/sso/login` sementara.
  - Versi B: Auth PMB (`/api/v1/auth/*`) tetap berjalan di atas Versi A.
  - Seeder service account (`pmb-pendaftaran`, `pmb-admin`, `pmb-mitra`, `keuangan`), migrasi data `user_roles` ke `user_accounts` + `user_account_roles`.
- **Status**: **done** (Fase K / M6–M8 selesai, implementasi SSO/OIDC standalone dan discovery endpoint).

### CR-013 — Kesiapan Rilis Aplikasi Pendaftaran PMB (4 Oktober 2026)
- **Arahan Lead**: Aplikasi pendaftaran camaba bebas bug dan siap didemokan Selasa 6 Okt malam, pembayaran sungguhan Duitku siap, jalur gratis tanpa pembayaran (Rp 0), panduan alur, data demo, dan audit stabilitas.
- **Ruang Lingkup**:
  - Audit mandiri dan perbaikan bug alur camaba (01–08).
  - Integrasi Duitku siap pakai (`DUITKU_ENV=sandbox|production`, `GET /finance/payment-config`, pengaman konfigurasi start, rekonsiliasi, callback idempoten).
  - Jalur gratis (`registration_fee = 0`): tagihan Rp 0 langsung lunas metode `free`, bypass pembayaran, `409 INVOICE_FREE` jika dibayar, pembatalan dan pindah jalur.
  - `scripts/smoke-camaba.sh`, `deploy/docker-compose.prod.yml`, `docs/BUGS.md`, `docs/DUITKU-PRODUKSI.md`, `docs/DEPLOY.md`, `docs/RELEASE-CHECKLIST.md`, `docs/PANDUAN-ALUR.md`.
- **Status**: **done** (Fase R / R0–R3 selesai, siap rilis v0.5.1-rilis).

### CR-014 — Kode Referal Guru BK, Pembayaran Kolektif & Komisi (4 Oktober 2026)
- **Arahan Lead**: Fitur kode referal guru BK untuk launching 22 Okt dengan 150 guru BK. Siswa bayar ke guru BK di luar sistem, verifikasi guru BK melewati pembayaran di sistem, guru BK menyetor ke kampus, komisi (persen/nominal).
- **Ruang Lingkup**:
  - Modul `referral` di `pmb-service`, role `guru_bk` di service account `pmb-mitra`.
  - Pemasangan kode oleh camaba dengan consent, verifikasi guru BK (`referral_collective`), penolakan, kunci kode setelah bayar.
  - Pengaturan komisi per kode (persen / nominal) disalin snapshot saat verifikasi; jalur gratis tanpa komisi; mode penyelesaian komisi (`terpisah` atau `potong_setoran`).
  - Pengajuan setoran guru BK dengan bukti transfer, verifikasi setoran oleh keuangan + pencatatan buku besar.
  - Portal & kartu ringkasan guru BK mirip sistem BTH, form pembuatan akun guru BK oleh admin + impor CSV multi-baris.
- **Status**: **done** (Fase F / F1–F5 selesai, modul terisolasi & teruji).

### CR-015 — Penyesuaian Launching (13 Oktober 2026)
- **Arahan Lead**: Penyesuaian menjelang launching sistem PMB meliputi pembayaran langsung Duitku v2 tanpa popup, WhatsApp group onboarding dengan gatekeeper, format NIM SK Rektor 065, validasi & ekspor biodata format SEVIMA (62 kolom), audit hardcode kampus, dan dashboard guru BK.
- **Ruang Lingkup**:
  - G1: Duitku API v2 inquiry langsung (tanpa popup), render QRIS base64 PNG dengan `go-qrcode`, instruksi per kanal.
  - G2: WhatsApp Group Onboarding (`GET/POST /admission/me/onboarding`), gatekeeper `409 WA_GROUP_JOIN_REQUIRED` pada aksi pendaftaran.
  - G3: Format NIM otomatis SK Rektor 065 (`{FAKULTAS}{PRODI}{KKNI}{JALUR}{TAHUN2}{SEMESTER}{URUT:3}`).
  - G4: Master referensi biodata & ekspor Excel SEVIMA 62-kolom dengan validasi tipe string cell (NIK, KK, HP, dsb).
  - G5: Identitas institusi terpusat & dinamis (`GET /organization/institution-profile`).
  - G6: Dashboard guru BK siap sosialisasi (`GET /api/v1/referral/partner/dashboard`).
- **Status**: **done** (Fase G / G1–G6 selesai, 100% tes lulus).



