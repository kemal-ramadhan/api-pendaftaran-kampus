# PRD — Sistem PMB Fase 2: Layanan Inti Pendaftaran s/d NIM

| | |
|---|---|
| **Produk** | Sistem Pendaftaran Mahasiswa Baru (PMB), bagian dari "Sistem Kampus" |
| **Lingkup** | Seluruh service setelah Auth & Gateway: Organization, Identity, Notification, Admission, Finance, Exam, Scoring, Selection, Student, NIM |
| **Prasyarat** | Fase 1 selesai: Auth Service + API Gateway (lulus e2e 43/43) |
| **Stack** | Sama dengan fase 1: Go, Gin, GORM + PostgreSQL 15, Redis 7, Docker Compose, + MinIO (penyimpanan file) |
| **Dokumen acuan** | `docs/flowchart-sistem-pmb.pdf`, `docs/use-case-sistem-pmb.pdf`, `docs/class-diagram-auth-service.md`, `PRD.md` (fase 1), `docs/DECISIONS.md` |
| **Status** | Draft v2 — ditambah fitur OCR KTP + peningkatan kualitas gambar (permintaan lead) |

---

### Perubahan v2
- Fitur baru: **scan KTP dengan OCR** saat pengisian biodata, dengan **peningkatan kualitas gambar berbasis AI** bila foto buram atau kurang jelas (Bagian 4.11, milestone F2-M2b).
- Service baru: **OCR Service** (Python), hanya internal.

## 0. Instruksi untuk AI Agent

1. PRD fase 1 (`PRD.md`) **tetap berlaku** untuk semua aturan umum: format respons (5.6), error code, keamanan, struktur folder, dan cara kerja per milestone. PRD ini hanya menambah dan tidak mengubahnya, kecuali disebut eksplisit.
2. Kerjakan milestone di **Bagian 9** secara berurutan. Satu milestone = satu service (atau satu bagian service). Di akhir setiap milestone: `go build`, `go vet`, `go test`, `docker compose up -d --build` dari database kosong, `scripts/e2e.sh` semua PASS, commit, lalu laporkan dan **tunggu persetujuan**.
3. **Jangan pernah mengubah perilaku Auth Service dan Gateway yang sudah lulus e2e**, kecuali untuk menambah route di `routes.yaml`, menambah endpoint internal yang diminta PRD ini, atau memperbaiki bug. Setiap perubahan pada kedua service itu wajib tetap meluluskan seluruh e2e fase 1.
4. Jika PRD ini ambigu, pilih solusi paling sederhana yang konsisten dengan pola fase 1, lalu catat di `docs/DECISIONS.md`. Pertanyaan terbuka yang sudah diberi default ada di Bagian 11; gunakan default-nya, jangan menunggu jawaban.
5. Semua nama tabel, field JSON, dan identifier kode dalam bahasa Inggris. Pesan untuk user dalam bahasa Indonesia.

---

## 1. Tujuan

Menjadikan seluruh alur PMB di flowchart dapat dijalankan end-to-end lewat API Gateway:

> Registrasi → pilih gelombang, jalur, prodi → bayar pendaftaran → isi biodata & upload dokumen → verifikasi dokumen → kartu ujian → seleksi (tes/wawancara/berkas) → pengumuman → bayar daftar ulang (lunas/cicilan) → daftar ulang → NIM & akun mahasiswa.

Beserta sisi internal: admin PMB, keuangan, dosen, prodi, pimpinan, dan superadmin sesuai use case diagram.

### Non-tujuan fase 2
- Integrasi payment gateway **produksi** (Midtrans/Xendit/bank). Fase ini memakai adapter **mock** di balik interface, adapter nyata menyusul.
- Pengiriman WhatsApp nyata (adapter disiapkan, driver default `log`).
- Ujian online berbasis soal (CBT). Exam service hanya mengelola jadwal, peserta, kartu, dan kehadiran; nilai diinput dosen.
- Sistem akademik setelah mahasiswa aktif (KRS, perkuliahan).
- Frontend.

---

## 2. Peta Service & Kepemilikan Data

Prinsip dari fase 1 tetap: **satu service memiliki datanya sendiri**, satu database per service di satu instance Postgres, tidak ada service yang membaca database service lain. `user_id` (UUID dari Auth) adalah ID global orang.

| Service | Prefix gateway | Pemilik data | Keterangan |
|---|---|---|---|
| Auth *(fase 1)* | `/api/v1/auth` | Akun, token, role, scope | Tambahan kecil: lihat 4.10 |
| Gateway *(fase 1)* | — | — | Tambah route per service |
| **Organization** | `/api/v1/organization` | Fakultas, program studi | Master data kampus, dibaca banyak service |
| **Identity** | `/api/v1/identity` | Biodata orang, alamat, orang tua/wali, riwayat pendidikan, foto profil | Data personal dipisah dari Auth (arahan lead) |
| **Notification** | *(internal saja)* | Template & log notifikasi | Tambahan dari tim, lihat 4.3 |
| **Admission** | `/api/v1/admission` | Gelombang, jalur, kuota, persyaratan dokumen, **pendaftaran (application)**, dokumen & verifikasinya | Inti PMB |
| **Finance** | `/api/v1/finance` | Skema biaya, tagihan, pembayaran, cicilan, kwitansi | |
| **Exam** | `/api/v1/exam` | Jadwal seleksi, sesi, peserta, dosen penguji, kartu ujian, kehadiran | |
| **Scoring** | `/api/v1/scoring` | Komponen & bobot penilaian, nilai, rekap, ranking | |
| **Selection** | `/api/v1/selection` | Rekomendasi kelulusan, persetujuan, pengumuman | |
| **Student** | `/api/v1/student` | Daftar ulang, data mahasiswa baru | |
| **NIM** | `/api/v1/nim` | Format & urutan NIM, riwayat penerbitan | |
| **OCR** *(v2)* | *(internal saja)* | Tidak menyimpan data permanen; hanya job OCR sementara | Scan KTP + peningkatan kualitas gambar, lihat 4.11 |

**Aturan akses umum semua service baru:**
- Wajib middleware `pkg/gatewayauth` (verifikasi signature gateway) untuk semua route kecuali `/health*` dan `/internal/*`.
- Identitas user dibaca dari header `X-User-ID`, `X-User-Roles`, `X-User-Scopes` yang sudah diverifikasi gateway. Service **tidak** memvalidasi JWT ulang, kecuali memang dibutuhkan.
- Role guard kasar di gateway (`routes.yaml`), otorisasi detail di service. Contoh: role `prodi` hanya melihat data prodi pada scope-nya; `calon_mahasiswa` hanya melihat pendaftarannya sendiri.
- Endpoint internal `/internal/*` memakai `X-Internal-Token` dan tidak diekspos gateway.

---

## 3. Keterlacakan Use Case → Service

| UC | Use case | Service | Status |
|---|---|---|---|
| UC-01 | Login | Auth | Selesai (fase 1) |
| UC-02 | Registrasi Akun | Auth | Selesai |
| UC-03 | Verifikasi Email / OTP | Auth | Selesai |
| UC-04 | Pilih Gelombang & Jalur | Admission | Selesai (Tahap 3) |
| UC-05 | Pilih Program Studi | Admission (+ Organization) | Selesai (Tahap 3) |
| UC-06 | Isi Biodata | Identity | Selesai |
| UC-06a *(baru)* | Scan KTP (OCR + perbaikan kualitas foto), «extend» Isi Biodata | Identity + OCR | Selesai |
| UC-07 | Upload Dokumen Persyaratan | Admission | Selesai (Tahap 5) |
| UC-08 | Bayar Biaya Pendaftaran | Finance | Selesai (Tahap 4) |
| UC-09 | Cetak Kartu Ujian | Exam | Selesai (Tahap 6) |
| UC-10 | Lihat Pengumuman Hasil | Selection | Selesai (Tahap 8) |
| UC-11 | Bayar Daftar Ulang | Finance | Selesai (Tahap 9) |
| UC-12 | Daftar Ulang | Student | Selesai (Tahap 10) |
| UC-13 | Proses Pembayaran | Finance | Selesai (Tahap 4) |
| UC-14 | Upload Bukti Transfer | Finance | Selesai (Tahap 4) |
| UC-15 | Ajukan Cicilan | Finance | Selesai (Tahap 9) |
| UC-16 | Generate NIM | NIM | Selesai (Tahap 11) |
| UC-17 | Kirim Notifikasi | Notification | Selesai (Tahap 3) |
| UC-18 | Kelola Data Pendaftar | Admission | Selesai (Tahap 3 & 5) |
| UC-19 | Verifikasi Dokumen | Admission | Selesai (Tahap 5) |
| UC-20 | Kelola Jadwal Seleksi | Exam | Selesai (Tahap 6) |
| UC-21 | Assign Dosen Penguji | Exam | Selesai (Tahap 6) |
| UC-22 | Publikasikan Pengumuman | Selection | Selesai (Tahap 8) |
| UC-23 | Verifikasi Pembayaran | Finance | Selesai (Tahap 4) |
| UC-24 | Setujui Pengajuan Cicilan | Finance | Selesai (Tahap 9) |
| UC-25 | Lihat Laporan Keuangan | Finance | Selesai (Tahap 9) |
| UC-26 | Export Laporan | Finance, Scoring, Admission, Student | Sebagian (Admission, Scoring, Finance, & Student export selesai di Tahap 5, 7, 9, 10) |
| UC-27 | Input Nilai Seleksi | Scoring | Selesai (Tahap 7) |
| UC-28 | Lihat Rekap & Ranking | Scoring | Selesai (Tahap 7) |
| UC-29 | Rekomendasi Kelulusan | Selection | Selesai (Tahap 8) |
| UC-30 | Setujui Kelulusan | Selection | Selesai (Tahap 8) |
| UC-31 | Lihat Dashboard Statistik | Admission + Finance (endpoint statistik) | Sebagian (Admission stats & Finance summary selesai) |
| UC-32 | Kelola Master Data | Organization (prodi selesai), Admission (gelombang, jalur, kuota, dokumen), Finance (biaya) | Sebagian (prodi selesai) |
| UC-33 | Kelola User & Role | Auth | Selesai |

---

## 4. Spesifikasi per Service

Setiap service mengikuti struktur `auth-service` (cmd/internal, repository, dto, migrations SQL, swagger, Dockerfile dengan BuildKit cache, `.env.example`). Skema di bawah adalah **minimum**; boleh ditambah kolom teknis (`created_at`, `updated_at`, `deleted_at`, `created_by`) di semua tabel.

### 4.1 Organization Service

**Tabel**
- `faculties`: `id`, `code` (unik, mis. `FT`), `name`, `is_active`.
- `study_programs`: `id`, `faculty_id`, `code` (unik, dipakai sebagai `scope_id` role prodi, mis. `TI`), `nim_code` (2 digit, untuk format NIM), `name`, `degree` (`D3`, `S1`, `S2`), `accreditation`, `is_active`.

**Endpoint**
| Method | Path | Akses |
|---|---|---|
| GET | `/faculties`, `/study-programs` (filter `faculty_code`, `is_active`) | Publik (dipakai form pendaftaran) |
| GET | `/study-programs/:code` | Publik |
| POST/PUT/PATCH | `/admin/faculties[/:id]`, `/admin/study-programs[/:id]` | `superadmin` |
| GET | `/internal/study-programs/:code`, `POST /internal/study-programs/batch` | Internal |

**Tambahan:** Auth Service mulai memvalidasi `scope_id` role `prodi` lewat `GET /internal/study-programs/:code` saat superadmin mengatur role (lihat 4.10).

Seeder development: 2 fakultas dan 4 prodi contoh.

### 4.2 Identity Service

Data personal, dipisah dari Auth sesuai arahan lead.

**Tabel**
- `persons`: `user_id` (PK, UUID dari Auth), `full_name`, `nik` (16 digit, unik, disimpan terenkripsi at-rest dengan AES-GCM dari env `IDENTITY_ENCRYPTION_KEY`, plus kolom `nik_hash` untuk pencarian unik), `birth_place`, `birth_date`, `gender`, `religion`, `nationality`, `phone`, `photo_object_key`.
- `addresses`: `user_id`, `type` (`ktp`, `domisili`), `street`, `rt`, `rw`, `village`, `district`, `city`, `province`, `postal_code`.
- `guardians`: `user_id`, `relation` (`ayah`, `ibu`, `wali`), `name`, `occupation`, `income_range`, `phone`.
- `education_histories`: `user_id`, `level` (`SMA`, `SMK`, `MA`, `D3`, …), `school_name`, `npsn`, `major`, `graduation_year`.

**Endpoint**
| Method | Path | Akses |
|---|---|---|
| GET/PUT | `/me/profile`, `/me/addresses`, `/me/guardians`, `/me/education` | User pemilik data |
| PUT | `/me/photo` (multipart, jpg/png maks 2 MB, disimpan di MinIO) | User |
| GET | `/me/completeness` → `{ "complete": bool, "missing": [..] }` | User |
| GET | `/admin/persons/:user_id` | `admin_pmb`, `superadmin` |
| GET | `/internal/persons/:user_id`, `/internal/persons/:user_id/completeness`, `POST /internal/persons/batch` | Internal |

Aturan kelengkapan biodata (dipakai Admission sebelum submit): profil, alamat KTP, minimal satu orang tua/wali, dan riwayat pendidikan terakhir wajib terisi.

NIK tidak pernah dikembalikan utuh ke role selain pemilik dan `admin_pmb`/`superadmin`; untuk role lain dimasking (`3205********0001`).

### 4.3 Notification Service *(tambahan tim)*

Satu tempat untuk kirim email/WA dan template, agar service lain tidak mengurus SMTP sendiri.

- Hanya endpoint internal: `POST /internal/notifications` body `{ "user_id" | "email", "channel": ["email","whatsapp"], "template": "document_verified", "data": {..}, "idempotency_key": "..." }`.
- Tabel `notification_logs`: status `queued`, `sent`, `failed`, `retries`, `error`.
- Pengiriman asinkron lewat antrean Redis (list/stream) + worker di service yang sama, retry 3 kali dengan backoff.
- Driver email: `smtp` / `log` (Mailpit di development). Driver WA: interface `WhatsAppSender` dengan driver `log` saja di fase ini.
- Template (HTML + teks, bahasa Indonesia): `payment_received`, `document_verified`, `document_revision`, `exam_card_ready`, `announcement_published`, `installment_reminder`, `reregistration_completed`, `nim_issued`.
- Auth Service **tidak** wajib dimigrasikan ke Notification Service di fase ini (sudah stabil). Catat sebagai pekerjaan lanjutan.

### 4.4 Admission Service

**Master data (UC-32)**
- `periods` (gelombang): `id`, `academic_year` (mis. `2026/2027`), `name` (mis. `Gelombang 1`), `registration_start`, `registration_end`, `exam_date_start`, `exam_date_end`, `announcement_date`, `reregistration_end`, `is_active`.
- `admission_paths` (jalur): `id`, `code` (`REGULER`, `PRESTASI`, `BEASISWA`, `PINDAHAN`), `name`, `description`, `selection_method` (`test`, `interview`, `document`, `test_interview`), `is_active`.
- `period_paths`: gelombang mana membuka jalur mana.
- `quotas`: `period_id`, `path_id`, `study_program_code`, `quota`.
- `document_requirements`: `path_id`, `code` (`IJAZAH_SKL`, `KTP`, `KK`, `PAS_FOTO`, `RAPOR`, `SERTIFIKAT_PRESTASI`, `TRANSKRIP`, `SURAT_PINDAH`), `name`, `is_required`, `allowed_types`, `max_size_mb`.

**Pendaftaran**
- `applications`: `id`, `registration_number` (format `PMB-{YYYY}-{period}-{urut 5 digit}`), `user_id`, `period_id`, `path_id`, `first_choice_program`, `second_choice_program`, `status`, `submitted_at`, `verified_at`, `verified_by`, `notes`.
- `application_documents`: `application_id`, `requirement_code`, `object_key`, `file_name`, `mime_type`, `size`, `status` (`uploaded`, `valid`, `invalid`), `review_note`, `reviewed_by`, `reviewed_at`.
- `application_status_logs`: riwayat perubahan status (siapa, kapan, dari → ke, catatan).

**Status pendaftaran (state machine, wajib divalidasi di service):**

```
draft ──(pilih gelombang/jalur/prodi, tagihan dibuat)──> awaiting_payment
awaiting_payment ──(Finance: tagihan pendaftaran lunas)──> form_open
form_open ──(biodata lengkap + dokumen wajib terunggah, submit)──> submitted
submitted ──(admin)──> verified | revision_required
revision_required ──(upload ulang + submit)──> submitted
verified ──(Exam: dijadwalkan)──> exam_scheduled
exam_scheduled ──(Selection: pengumuman terbit)──> passed | failed
passed ──(Student: daftar ulang selesai)──> reregistered
reregistered ──(NIM terbit)──> enrolled
```

Aturan: satu user hanya boleh punya **satu pendaftaran aktif per gelombang**. Perubahan prodi hanya boleh saat `draft`/`awaiting_payment`. Pendaftaran tidak bisa dibuat di luar rentang `registration_start`–`registration_end`.

**Endpoint utama**
| Method | Path | Akses |
|---|---|---|
| GET | `/periods/active`, `/periods/:id/paths`, `/paths/:id/requirements` | Publik |
| POST | `/applications` (pilih gelombang, jalur, prodi 1 & 2) | `calon_mahasiswa` |
| GET | `/applications/me` | `calon_mahasiswa` |
| PUT | `/applications/me/choices` | `calon_mahasiswa` (status `draft`/`awaiting_payment`) |
| POST | `/applications/me/documents` (multipart) | `calon_mahasiswa` |
| DELETE | `/applications/me/documents/:code` | `calon_mahasiswa` (sebelum submit) |
| POST | `/applications/me/submit` | `calon_mahasiswa` |
| GET | `/admin/applications` (filter status, gelombang, jalur, prodi, q; paginasi) | `admin_pmb`, `superadmin`, `prodi` (hanya prodinya), `pimpinan` (read-only) |
| GET | `/admin/applications/:id` (+ biodata dari Identity, status bayar dari Finance) | sama |
| GET | `/admin/applications/:id/documents/:code/url` (presigned URL 5 menit) | `admin_pmb`, `superadmin` |
| POST | `/admin/applications/:id/verify` body per dokumen `valid`/`invalid` + catatan | `admin_pmb` |
| GET | `/admin/applications/export` (xlsx) | `admin_pmb`, `superadmin`, `pimpinan` |
| CRUD | `/admin/periods`, `/admin/paths`, `/admin/quotas`, `/admin/requirements` | `superadmin` (baca juga `admin_pmb`) |
| GET | `/admin/stats` (jumlah per status, per prodi, per jalur, per gelombang; konversi pendaftar → daftar ulang) | `pimpinan`, `admin_pmb`, `superadmin` |
| GET/POST | `/internal/applications/:id`, `/internal/applications/:id/status` | Internal |

Setiap perubahan status penting memanggil Notification Service (verifikasi, revisi).

**Penyimpanan dokumen:** MinIO (S3-compatible) di docker-compose. Bucket privat per service (`admission-documents`). File tidak pernah diakses publik; hanya lewat presigned URL berumur pendek. Validasi MIME dari isi file (bukan ekstensi), ukuran maks per requirement (default 2 MB, PDF/JPG/PNG).

### 4.5 Finance Service

**Tabel**
- `fee_schemes`: `period_id`, `path_id`, `study_program_code` (nullable = berlaku semua prodi), `fee_type` (`registration`, `reregistration`), `amount` (integer rupiah, **jangan float**), `installment_allowed`, `max_installments`, `min_first_payment_percent`.
- `invoices`: `id`, `invoice_number` (`INV-{YYYYMMDD}-{urut}`), `user_id`, `application_id`, `fee_type`, `amount`, `paid_amount`, `status` (`unpaid`, `partially_paid`, `paid`, `cancelled`, `expired`), `due_date`, `payment_scheme` (`full`, `installment`).
- `installment_requests`: `invoice_id`, `requested_terms`, `reason`, `status` (`pending`, `approved`, `rejected`), `reviewed_by`, `review_note`.
- `installments`: `invoice_id`, `term_no`, `amount`, `due_date`, `status`, `paid_at`.
- `payments`: `id`, `invoice_id`, `installment_id` (nullable), `method` (`virtual_account`, `manual_transfer`), `provider`, `provider_ref`, `va_number`, `amount`, `status` (`pending`, `success`, `failed`, `expired`, `awaiting_verification`, `rejected`), `proof_object_key`, `verified_by`, `verified_at`, `paid_at`.
- `receipts`: `payment_id`, `receipt_number`, `pdf_object_key`.
- `payment_webhook_logs`: payload mentah + hasil verifikasi signature (untuk audit).

**Aturan**
- Tagihan pendaftaran dibuat otomatis saat Admission membuat pendaftaran (panggilan internal `POST /internal/invoices`). Pendaftaran selalu `full`.
- Tagihan daftar ulang dibuat saat Selection menerbitkan status `passed`.
- Cicilan hanya untuk `reregistration` dan hanya bila `installment_allowed`. Default: maks 4 termin, pembayaran pertama minimal 25%, jatuh tempo per 30 hari.
- **Payment provider** lewat interface `PaymentProvider` (`CreateVA`, `ParseWebhook`, `VerifySignature`). Fase ini: `MockProvider` + endpoint simulasi bayar khusus development `POST /dev/payments/:id/simulate-paid` (hanya aktif bila `APP_ENV=development`).
- Webhook provider: route publik `POST /webhooks/payments/:provider` di gateway (tanpa JWT), diverifikasi dengan signature provider, **idempoten** berdasarkan `provider_ref`.
- Transfer manual: user unggah bukti → status `awaiting_verification` → keuangan setujui/tolak.
- Saat tagihan pendaftaran lunas → panggil Admission `POST /internal/applications/:id/status` ke `form_open`. Saat daftar ulang lunas atau termin pertama dibayar → beri tahu Student Service bahwa daftar ulang boleh dilanjutkan.
- Semua perhitungan uang memakai integer; total `payments` sukses tidak boleh melebihi `amount` tagihan.
- Job terjadwal (goroutine ticker atau cron ringan di service): reminder termin H-3 dan H-1 via Notification; menandai tagihan `expired` sesuai `due_date`.

**Endpoint utama**
| Method | Path | Akses |
|---|---|---|
| GET | `/invoices/me`, `/invoices/me/:id` | pemilik |
| POST | `/invoices/me/:id/pay` body `{ "method": "virtual_account" }` → nomor VA | pemilik |
| POST | `/invoices/me/:id/manual-transfer` (multipart bukti) | pemilik |
| POST | `/invoices/me/:id/installment-request` | pemilik (tagihan daftar ulang) |
| GET | `/receipts/me/:payment_id` (PDF) | pemilik |
| GET | `/admin/payments?status=awaiting_verification` | `keuangan` |
| POST | `/admin/payments/:id/verify` (`approve`/`reject` + catatan) | `keuangan` |
| GET | `/admin/installment-requests`, `POST /admin/installment-requests/:id/review` | `keuangan` |
| CRUD | `/admin/fee-schemes` | `superadmin`, `keuangan` |
| GET | `/admin/reports/summary` (pemasukan per periode/jalur/prodi, tunggakan) | `keuangan`, `pimpinan`, `superadmin` |
| GET | `/admin/reports/export?format=xlsx` | `keuangan`, `pimpinan`, `superadmin` |
| POST | `/webhooks/payments/:provider` | Publik (signature provider) |
| POST | `/internal/invoices`, `GET /internal/invoices?application_id=` | Internal |

### 4.6 Exam Service

**Tabel**
- `exam_sessions`: `id`, `period_id`, `path_id`, `type` (`written_test`, `interview`), `study_program_code` (nullable), `start_at`, `end_at`, `location` atau `meeting_link`, `capacity`.
- `session_examiners`: `session_id`, `examiner_user_id` (role `dosen`).
- `participants`: `session_id`, `application_id`, `user_id`, `participant_number` (unik), `attendance` (`present`, `absent`, `null`).
- `exam_cards`: `participant_id`, `pdf_object_key`, `generated_at`.

**Aturan**
- Hanya pendaftaran berstatus `verified` yang bisa dijadwalkan. Setelah dijadwalkan, Exam memberi tahu Admission (`exam_scheduled`) dan Notification (`exam_card_ready`).
- Jalur dengan `selection_method = document` tidak perlu sesi ujian; langsung ke Scoring (penilaian berkas).
- Kartu ujian PDF berisi: nomor peserta, nama, foto (dari Identity), prodi pilihan, jadwal, lokasi/link, QR berisi nomor peserta.

**Endpoint utama**
| Method | Path | Akses |
|---|---|---|
| CRUD | `/admin/sessions` | `admin_pmb` |
| POST | `/admin/sessions/:id/examiners` | `admin_pmb` |
| POST | `/admin/sessions/:id/participants` (tambah manual atau auto-assign dari pendaftar `verified`) | `admin_pmb` |
| PATCH | `/admin/participants/:id/attendance` | `admin_pmb`, `dosen` (sesinya sendiri) |
| GET | `/me/schedule`, `/me/card` (PDF) | `calon_mahasiswa` |
| GET | `/examiner/sessions` (sesi yang ditugaskan) | `dosen` |

### 4.7 Scoring Service

**Tabel**
- `score_components`: `path_id`, `code` (`written_test`, `interview`, `report_card`, `achievement`), `name`, `weight_percent` (total per jalur = 100), `max_score`.
- `scores`: `application_id`, `component_code`, `score`, `input_by`, `input_at`, `note`.
- `score_summaries`: `application_id`, `final_score`, `rank_in_program`, `computed_at` (dihitung ulang saat nilai berubah).

**Aturan**
- Dosen hanya bisa input nilai untuk peserta di sesi yang ditugaskan kepadanya (cek ke Exam via internal API).
- Nilai tidak bisa diubah setelah Selection mengunci rekomendasi (flag `locked` per gelombang+prodi dari Selection).
- Ranking per prodi pilihan 1, per gelombang, per jalur.

**Endpoint utama**
| Method | Path | Akses |
|---|---|---|
| CRUD | `/admin/components` | `superadmin`, `admin_pmb` |
| GET | `/examiner/participants?session_id=` | `dosen` |
| PUT | `/examiner/scores/:application_id` | `dosen` |
| GET | `/program/recap?period_id=&path_id=` (rekap & ranking) | `prodi` (scope sendiri), `admin_pmb`, `pimpinan` |
| GET | `/program/recap/export` (xlsx) | sama |
| GET | `/internal/summaries?period_id=&program=` | Internal |

### 4.8 Selection Service

**Tabel**
- `recommendations`: `period_id`, `path_id`, `study_program_code`, `status` (`draft`, `submitted`, `approved`, `returned`), `submitted_by`, `approved_by`, `note`.
- `recommendation_items`: `recommendation_id`, `application_id`, `decision` (`pass`, `fail`, `waitlist`), `final_score`, `rank`.
- `announcements`: `period_id`, `title`, `published_at`, `published_by`.
- `results`: `application_id`, `user_id`, `decision`, `accepted_program`, `announced_at`.

**Alur (flowchart 4)**
1. Prodi melihat rekap (dari Scoring), menandai `pass`/`fail`/`waitlist` sesuai kuota (kuota dari Admission). Sistem menolak jumlah `pass` melebihi kuota.
2. Prodi submit rekomendasi → Scoring dikunci untuk kombinasi itu.
3. `pimpinan` menyetujui atau mengembalikan (dengan catatan) → bila dikembalikan, kembali ke prodi.
4. `admin_pmb` mempublikasikan pengumuman untuk rekomendasi yang `approved` → isi `results`, update status di Admission (`passed`/`failed`), minta Finance membuat tagihan daftar ulang untuk yang lulus, kirim notifikasi.
5. Hasil tidak terlihat oleh calon mahasiswa sebelum `published_at`.

**Endpoint utama**
| Method | Path | Akses |
|---|---|---|
| GET/PUT | `/program/recommendations` (per gelombang, jalur, prodi) | `prodi` |
| POST | `/program/recommendations/:id/submit` | `prodi` |
| GET | `/approval/recommendations`, `POST /approval/recommendations/:id/(approve\|return)` | `pimpinan` |
| POST | `/admin/announcements` (publikasi) | `admin_pmb` |
| GET | `/me/result` | `calon_mahasiswa` |

### 4.9 Student Service & NIM Service

**Student Service**
- `reregistrations`: `application_id`, `user_id`, `accepted_program`, `status` (`waiting_payment`, `in_progress`, `completed`), `completed_at`.
- `students`: `user_id`, `nim`, `study_program_code`, `entry_year`, `entry_path`, `status` (`active`).
- Syarat daftar ulang: hasil `passed`, tagihan daftar ulang lunas **atau** termin pertama cicilan terbayar, data tambahan terisi (ukuran almamater, konfirmasi data diri, pernyataan persetujuan). Setelah `completed` → panggil NIM Service.
- Endpoint: `GET /me/reregistration`, `PUT /me/reregistration` (data tambahan), `POST /me/reregistration/complete` (`calon_mahasiswa`); `GET /admin/students`, export xlsx (`admin_pmb`, `prodi` scope, `pimpinan`).

**NIM Service**
- `nim_formats`: template dapat dikonfigurasi. **Default:** `{YY}{nim_code_prodi}{jalur_digit}{urut 4}` → contoh `2655 1 0001` = `265510001` (YY = 2 digit tahun masuk, `nim_code` dari Organization, `jalur_digit` 1=reguler, 2=prestasi, 3=beasiswa, 4=pindahan).
- `nim_sequences`: `entry_year`, `study_program_code`, `last_number`, diperbarui dalam transaksi dengan `SELECT … FOR UPDATE` agar **tidak ada NIM ganda** walau request bersamaan.
- `nim_issuances`: riwayat penerbitan.
- Alur: `POST /internal/nim/issue` (dipanggil Student) → generate NIM → Student menyimpan ke `students` → Auth `POST /internal/users/:id/roles` menambah role `mahasiswa` → Admission status `enrolled` → notifikasi `nim_issued`.
- Endpoint admin: `GET /admin/issuances`, CRUD `/admin/formats` (`superadmin`).
- Idempoten: memanggil `issue` dua kali untuk `application_id` yang sama mengembalikan NIM yang sama.

### 4.10 Perubahan Kecil pada Auth & Gateway

- **Auth:** validasi `scope_id` role `prodi` ke Organization Service saat superadmin mengatur role. Jika Organization tidak tersedia, tolak dengan `UPSTREAM_UNAVAILABLE` (jangan simpan scope yang tidak tervalidasi).
- **Auth:** endpoint `POST /internal/users/:id/roles` dipakai NIM flow (sudah ada di fase 1; pastikan idempoten).
- **Gateway:** tambah route semua service di `routes.yaml` sesuai tabel akses di atas; tambah route publik webhook `POST /api/v1/finance/webhooks/payments/:provider`; body limit 5 MB untuk route upload (Identity foto, Admission dokumen, Finance bukti transfer).

---

### 4.11 OCR KTP & Peningkatan Kualitas Gambar *(v2)*

**Tujuan:** calon mahasiswa cukup memfoto atau mengunggah scan KTP, lalu data biodata (NIK, nama, tempat/tanggal lahir, jenis kelamin, alamat, RT/RW, kelurahan/desa, kecamatan, agama, kewarganegaraan) terisi otomatis. Bila foto buram, gelap, miring, atau beresolusi rendah, sistem memperbaiki kualitasnya dulu sebelum dibaca. Jika tetap tidak terbaca, user diminta foto ulang dengan petunjuk yang jelas.

**Prinsip yang tidak boleh dilanggar**
1. **Hasil OCR hanya saran, bukan data final.** Semua field tampil sebagai isian yang **wajib dikonfirmasi/diedit user** sebelum disimpan. Tidak ada auto-save.
2. **Peningkatan kualitas tidak boleh "mengarang" huruf/angka.** Dilarang memakai model generatif (mis. GAN super-resolution) yang bisa menciptakan karakter yang tidak ada. Hanya teknik yang mempertahankan isi: koreksi perspektif, deskew, denoise, sharpening, kontras adaptif (CLAHE), binarisasi, dan super-resolution ringan non-generatif (mis. OpenCV DNN FSRCNN/ESPCN).
3. **Gambar asli adalah dokumen resmi.** Gambar hasil perbaikan hanya dipakai untuk OCR dan tidak menggantikan file asli yang diverifikasi admin.
4. **Data KTP adalah data pribadi (UU No. 27/2022 tentang Pelindungan Data Pribadi).** Proses default berjalan **lokal di server sendiri**. OCR Service tidak menyimpan gambar maupun hasil secara permanen, dan log tidak boleh memuat NIK atau gambar. Penyedia AI/OCR cloud hanya boleh diaktifkan lewat konfigurasi, dengan persetujuan eksplisit user dan persetujuan lead (default: **mati**).

**Arsitektur**
- **OCR Service** ditulis dengan **Python 3.12 + FastAPI**, karena ekosistem OCR/computer vision (OpenCV, PaddleOCR) jauh lebih matang di Python. Ini satu-satunya service non-Go; tetap mengikuti kontrak fase 1: format respons 5.6, `X-Internal-Token`, `X-Request-ID`, healthcheck, Dockerfile non-root, `.env.example`.
- **Tidak diekspos gateway.** Hanya dipanggil Identity Service lewat `/internal/*`.
- Engine lewat interface `OcrEngine` (default `PaddleOCR`, cadangan `Tesseract` dengan bahasa `ind`) dan `ImageEnhancer`. Adapter cloud (`CloudVisionEngine`) disiapkan tetapi dimatikan (`OCR_CLOUD_ENABLED=false`).
- Pemrosesan **asinkron**: job masuk antrean Redis, worker memproses, hasil disimpan sementara di Redis dengan TTL 15 menit lalu terhapus otomatis.

**Pipeline**
1. **Validasi input:** JPG/PNG/PDF (halaman 1), maks 5 MB, resolusi minimal 640 px sisi terpendek.
2. **Deteksi & crop kartu:** temukan kontur kartu, koreksi perspektif, luruskan kemiringan, normalisasi orientasi.
3. **Cek kualitas** → skor 0–100 dari: ketajaman (variansi Laplacian), kecerahan & kontras, silau/pantulan, resolusi efektif, dan persentase kartu di frame.
4. **Perbaikan bila skor < ambang (default 70):** denoise, CLAHE, sharpening, koreksi pencahayaan, reduksi silau, dan super-resolution non-generatif bila resolusi rendah. Hitung ulang skor.
5. **OCR** pada gambar terbaik (asli atau hasil perbaikan), per region field KTP.
6. **Parsing & validasi:**
   - NIK 16 digit; koreksi salah baca yang umum hanya pada posisi angka (`O→0`, `I/l→1`, `B→8`, `S→5`).
   - Konsistensi NIK: digit 7–12 = tanggal lahir DDMMYY (perempuan: tanggal + 40) harus cocok dengan tanggal lahir dan jenis kelamin hasil OCR; kode wilayah 6 digit awal harus valid (tabel kode wilayah Kemendagri di-seed ke OCR Service).
   - Normalisasi: huruf kapital, format tanggal `YYYY-MM-DD`, jenis kelamin → `L`/`P`.
7. **Hasil** per field: `value`, `confidence` (0–1), `needs_review` (true bila confidence < 0.85 atau gagal validasi konsistensi).

**Respons jika gagal:** status `retake_required` dengan alasan spesifik yang bisa ditampilkan frontend: `BLURRY` ("Foto buram, pegang HP lebih stabil"), `TOO_DARK`, `GLARE` ("Ada pantulan cahaya, hindari lampu langsung"), `CARD_NOT_FOUND` ("Pastikan seluruh KTP terlihat dalam bingkai"), `LOW_RESOLUTION`, `NOT_KTP`.

**Endpoint**

OCR Service (internal):
| Method | Path | Keterangan |
|---|---|---|
| POST | `/internal/ocr/ktp` (multipart) | Buat job → `{ "job_id": "..." }` |
| GET | `/internal/ocr/jobs/:id` | `processing` / `completed` / `retake_required` / `failed` + hasil |
| GET | `/health`, `/health/ready` | |

Identity Service (lewat gateway, role `calon_mahasiswa`):
| Method | Path | Keterangan |
|---|---|---|
| POST | `/me/ktp-scan` (multipart, maks 5 MB) | Simpan gambar asli ke MinIO (bucket privat `identity-ktp`), kirim ke OCR → `{ "scan_id": "..." }` |
| GET | `/me/ktp-scan/:scan_id` | Status + field saran + `quality_score` + `enhanced` (bool) + alasan bila `retake_required` |
| POST | `/me/ktp-scan/:scan_id/confirm` | User mengirim field yang **sudah dicek/diedit** → disimpan ke `persons`/`addresses` |

Identity mencatat tabel `ktp_scans`: `id`, `user_id`, `object_key` (gambar asli), `status`, `quality_score`, `enhanced`, `field_confidences` (jsonb, **tanpa nilai NIK**), `confirmed_at`, `fields_edited_by_user` (daftar field yang diubah user dari saran OCR). Nilai hasil OCR mentah tidak disimpan permanen.

**Dukungan verifikasi admin:** di detail pendaftar (Admission), tampilkan gambar KTP asli, `quality_score`, dan tanda bila user mengubah field penting (NIK, nama, tanggal lahir) dari hasil OCR, agar admin lebih teliti saat verifikasi dokumen. Ini membantu, bukan menggantikan, verifikasi manual.

**Rate limit:** maksimal 10 scan per user per jam.

**Konfigurasi (`.env.example` OCR Service):** `OCR_ENGINE=paddle`, `OCR_QUALITY_THRESHOLD=70`, `OCR_FIELD_CONFIDENCE_THRESHOLD=0.85`, `OCR_RESULT_TTL=15m`, `OCR_MAX_FILE_MB=5`, `OCR_CLOUD_ENABLED=false`, `INTERNAL_TOKEN`, `REDIS_ADDR`, `WORKER_CONCURRENCY=2`.

**Pengujian**
- **Dilarang memakai foto KTP orang sungguhan** di repo maupun test. Buat **KTP sintetis** (generator gambar dengan data palsu dan layout menyerupai KTP) di `services/ocr-service/tests/fixtures/`, plus varian buram (Gaussian/motion blur), gelap, silau, miring, dan resolusi rendah.
- Target akurasi pada dataset sintetis: NIK **exact match ≥ 95%** untuk gambar jernih dan **≥ 85%** untuk gambar buram ringan setelah perbaikan; field nama & tanggal lahir ≥ 90%.
- Gambar yang sangat buram/terpotong harus menghasilkan `retake_required`, **bukan** hasil yang salah dengan confidence tinggi.
- Unit test validasi NIK (tanggal lahir, jenis kelamin, kode wilayah) dan koreksi karakter.
- Laporan akurasi otomatis `make ocr-benchmark` → `docs/ocr-benchmark.md` (akurasi per field, sebelum vs sesudah perbaikan kualitas, waktu proses rata-rata).
- e2e: upload KTP sintetis → polling hasil → confirm → biodata tersimpan; upload gambar sangat buram → `retake_required`.

**Kinerja:** target ≤ 5 detik per KTP di CPU (tanpa GPU) untuk gambar biasa. Model diunduh saat build image, bukan saat runtime.

## 5. Komunikasi Antar Service

- **Sinkron:** HTTP internal ke `/internal/*` dengan header `X-Internal-Token` dan `X-Request-ID` diteruskan. Buat klien bersama di `pkg/internalclient` (timeout 5 detik, retry 2 kali untuk GET, circuit sederhana opsional).
- **Perubahan status lintas service** (mis. Finance → Admission saat lunas): panggilan internal yang **idempoten** (menyertakan `idempotency_key`), dengan tabel **outbox** di service pengirim + worker retry. Tujuannya: bila service tujuan sedang mati, perubahan tidak hilang.
- Event bus (Redis Streams/Kafka) **tidak** dipakai di fase ini; catat sebagai pengembangan berikutnya.
- Service tidak saling memanggil secara melingkar dalam satu request. Bila perlu data gabungan (mis. detail pendaftar = Admission + Identity + Finance), service pemilik halaman (Admission) yang mengumpulkan lewat internal API.

## 6. Infrastruktur

- `deploy/docker-compose.yml`: tambah MinIO (+ inisialisasi bucket), setiap service baru, database per service (dibuat oleh init script Postgres).
- Setiap service punya healthcheck; gateway `depends_on` semua service yang dirutekan.
- `Makefile`: target per service (`make run-admission`, `make test-finance`, dst.) dan `make e2e`.
- Seeder development lintas service: 1 gelombang aktif, 4 jalur, kuota, persyaratan dokumen, skema biaya, komponen nilai, akun contoh untuk setiap role (password di README).

## 7. Testing

- Unit test per service (target coverage `service` ≥ 70%), termasuk: state machine pendaftaran (transisi ilegal ditolak), kuota, perhitungan cicilan (integer, jumlah termin = total), idempotensi webhook, NIM tanpa duplikasi (test konkurensi), otorisasi scope prodi, akses data milik sendiri.
- `scripts/e2e.sh` diperluas bertahap per milestone hingga menjalankan **seluruh flowchart 1** lewat gateway:
  register → verify → buat pendaftaran → bayar (simulate-paid) → isi biodata → upload dokumen → submit → admin verifikasi (satu kali revisi dulu) → jadwal & kartu → dosen input nilai → prodi rekomendasi → pimpinan setujui → admin publikasi → cek hasil → ajukan cicilan → keuangan setujui → bayar termin 1 → daftar ulang → NIM terbit → login ulang dan token memuat role `mahasiswa`.
- Skenario negatif minimal per service: akses data user lain (403/404), role salah (403), prodi di luar scope (403), transisi status ilegal (409 `INVALID_STATE`), kuota penuh (409 `QUOTA_EXCEEDED`), file bukan PDF/JPG/PNG atau terlalu besar (422), webhook signature salah (401), bypass gateway (401).
- Error code baru dicatat di `docs/API.md`: `INVALID_STATE`, `QUOTA_EXCEEDED`, `PERIOD_CLOSED`, `DUPLICATE_APPLICATION`, `DOCUMENT_REQUIRED`, `BIODATA_INCOMPLETE`, `PAYMENT_REQUIRED`, `INSTALLMENT_NOT_ALLOWED`, `SCORES_LOCKED`, `RESULT_NOT_PUBLISHED`, `FILE_TYPE_NOT_ALLOWED`, `FILE_TOO_LARGE`, `WEBHOOK_SIGNATURE_INVALID`.

## 8. Dokumentasi per Service

Setiap milestone wajib menghasilkan:
- Swagger service tersebut.
- Bagian baru di `docs/API.md` (endpoint, contoh request/response, error code).
- Folder baru di Postman collection.
- `docs/class-diagram-<service>.md` (Mermaid, dibuat dari kode aktual, pola yang sama dengan auth).
- ERD Mermaid di `docs/erd-<service>.md`.
- Update `docs/DECISIONS.md` dan tabel keterlacakan (Bagian 3) dengan status terbaru.

## 9. Milestone

| Milestone | Isi | Bukti selesai |
|---|---|---|
| **F2-M0** Fondasi bersama | `pkg/internalclient`, `pkg/storage` (MinIO), `pkg/pagination`, `pkg/outbox`, template service baru, MinIO & init DB di compose | **Selesai** (paket bersama lulus test, MinIO & basis data siap di compose) |
| **F2-M1** Organization | 4.1 + validasi scope di Auth (4.10) | **Selesai** (Organization Service selesai, superadmin membuat prodi, validasi scope prodi di Auth aktif, e2e terintegrasi) |
| **F2-M2** Identity | 4.2 | calon mahasiswa mengisi biodata; `completeness` benar; NIK terenkripsi di DB |
| **F2-M2b** OCR KTP *(v2)* | 4.11: OCR Service (Python) + endpoint scan di Identity | KTP sintetis terbaca otomatis, gambar buram diperbaiki atau diminta foto ulang, benchmark akurasi memenuhi target |
| **F2-M3** Notification | 4.3 | notifikasi masuk Mailpit, retry saat SMTP mati |
| **F2-M4** Admission bagian 1 | Master data + buat pendaftaran + state machine awal | pendaftaran dibuat, kuota & periode tervalidasi |
| **F2-M5** Finance bagian 1 | Skema biaya, tagihan, VA mock, webhook, transfer manual, verifikasi, kwitansi | bayar pendaftaran → Admission `form_open` |
| **F2-M6** Admission bagian 2 | Upload dokumen, submit, verifikasi & revisi, export, statistik | **Selesai** (alur upload dokumen, validasi magic bytes, submit, verifikasi admin, revisi, presigned URL 5 menit, export xlsx, statistik selesai) |
| **F2-M7** Exam | 4.6 | **Selesai** (jadwal sesi, dosen penguji, peserta verified, kartu PDF + QR, presensi selesai) |
| **F2-M8** Scoring | 4.7 | dosen input nilai, rekap & ranking prodi |
| **F2-M9** Selection | 4.8 | **Selesai** (rekomendasi prodi divalidasi kuota mutlak QUOTA_EXCEEDED, submit mengunci nilai di Scoring SCORES_LOCKED, persetujuan pimpinan approve/return, publikasi pengumuman oleh admin memutasi status Admission passed/failed, invoice daftar ulang Finance, notifikasi kelulusan, dan me result terproteksi RESULT_NOT_PUBLISHED) |
| **F2-M10** Finance bagian 2 | Cicilan, reminder, laporan & export | **Selesai** (pengajuan cicilan, larangan cicilan pendaftaran INSTALLMENT_NOT_ALLOWED, pembagian termin presisi integer tanpa selisih pembulatan, persetujuan staf keuangan, pembayaran per termin, transisi status partially_paid/paid, cron reminder H-3 & H-1 via notification service, penandaan expired, ringkasan laporan & ekspor Excel .xlsx) |
| **F2-M11** Student + NIM | 4.9 | **Selesai** (bagian Student selesai di Tahap 10: formulir daftar ulang, validasi kelulusan passed & kliring pembayaran lunas/termin 1 partially_paid, tabel students berstatus reregistered, outbox penerbitan NIM, isolasi prodi, ekspor spreadsheet Excel .xlsx; bagian NIM selesai di Tahap 11: NIM Service, template format dinamis, locking pesimistik SELECT FOR UPDATE tanpa NIM ganda, idempotensi application_id, worker publisher outbox otomatis di Student Service, orkestrasi mutasi student active, auth role mahasiswa, admission enrolled, notifikasi nim_issued) |
| **F2-M12** Integrasi akhir | e2e flowchart 1 penuh, dashboard statistik pimpinan, Postman & dokumentasi lengkap | `e2e.sh` seluruh alur PASS dari database kosong |

### 9.1 Pemetaan ke Timeline Tim

Pekerjaan dieksekusi **bertahap mengikuti baris timeline** (sheet Timeline Development IT). Satu tahap = satu atau dua milestone di atas.

| Tahap | Baris timeline | Milestone | Rencana tanggal |
|---|---|---|---|
| 1 | Organization Service | F2-M0 + F2-M1 | 22–25 Sep 2026 |
| 2 | Identity Service | F2-M2 + F2-M2b (OCR KTP) | 26 Sep – 6 Okt 2026 |
| 3 | Admission Service (1) | F2-M3 (Notification) + F2-M4 | 7–13 Okt 2026 |
| 4 | Finance Service (bagian 1) | F2-M5 | 14–20 Okt 2026 |
| 5 | Admission Service (2) | F2-M6 | 21–26 Okt 2026 |
| 6 | Exam Service | F2-M7 | 27 Okt – 2 Nov 2026 |
| 7 | Scoring Service | F2-M8 | 3–6 Nov 2026 |
| 8 | Selection Service | F2-M9 | 7–12 Nov 2026 |
| 9 | Finance Service (bagian 2) | F2-M10 | 13–18 Nov 2026 |
| 10 | Student Service | F2-M11 (bagian Student) | 19–21 Nov 2026 |
| 11 | NIM Service | F2-M11 (bagian NIM) + F2-M12 | 22–30 Nov 2026 |

Tambahan atau perubahan dari lead dimasukkan sebagai **Change Request** (lihat `docs/CHANGE_REQUESTS.md`) dan dikerjakan di tahap yang paling relevan tanpa merombak tahap yang sudah selesai.

Urutan boleh disesuaikan dengan sprint di timeline tim; jika berubah, catat di `docs/DECISIONS.md`. Prinsip yang tidak boleh dilanggar: service yang menjadi ketergantungan dikerjakan lebih dulu.

## 10. Definition of Done Fase 2

- [ ] Seluruh use case UC-01 s/d UC-33 berstatus **Selesai** di tabel keterlacakan.
- [ ] Alur flowchart 1–5 dapat dijalankan end-to-end lewat gateway dan tercakup di `scripts/e2e.sh`.
- [ ] Setiap service: signature gateway wajib, otorisasi per role & scope, test negatif lulus.
- [ ] Tidak ada service yang membaca database service lain.
- [ ] Uang disimpan sebagai integer; NIM dijamin unik; webhook idempoten.
- [ ] Dokumen dan foto hanya dapat diakses lewat presigned URL berumur pendek.
- [ ] Dokumentasi, Postman, class diagram, dan ERD setiap service tersedia dan sesuai kode.
- [ ] `docker compose up -d --build` dari clone bersih + `scripts/e2e.sh` semua PASS.
- [ ] OCR KTP memenuhi target akurasi di `docs/ocr-benchmark.md`, tidak menyimpan gambar/hasil secara permanen di OCR Service, dan tidak mengirim data ke pihak ketiga secara default.

## 11. Pertanyaan Terbuka (dipakai default dulu)

| Pertanyaan | Default sementara |
|---|---|
| Metode seleksi per jalur | Reguler: tes tulis + wawancara; Prestasi: berkas + wawancara; Beasiswa: tes tulis + wawancara; Pindahan: berkas. Dapat diubah via master jalur & komponen nilai |
| Aturan cicilan | Hanya daftar ulang, maks 4 termin, minimal 25% di termin pertama, jatuh tempo per 30 hari, tanpa denda |
| Vendor payment gateway | Mock provider; interface siap untuk Midtrans/Xendit |
| Dokumen wajib per jalur | Semua: KTP, KK, pas foto, ijazah/SKL. Reguler & Beasiswa: + rapor. Prestasi: + sertifikat prestasi. Pindahan: + transkrip & surat pindah |
| Siapa yang menyetujui kelulusan | Role `pimpinan` |
| Format & sumber NIM | Dibuat NIM Service dengan format default 4.9, dapat diubah superadmin |
| WhatsApp | Adapter `log`, belum mengirim sungguhan |
| Pembayaran pendaftaran bisa refund? | Tidak ada refund di sistem; dicatat manual oleh keuangan |
| OCR KTP: boleh memakai layanan AI/OCR cloud? | Tidak; proses lokal (PaddleOCR + OpenCV). Adapter cloud disiapkan tapi mati, perlu persetujuan lead + persetujuan user |
| OCR KTP: wajib atau opsional? | Opsional; user tetap bisa isi biodata manual |
| Server punya GPU? | Diasumsikan tidak; pipeline dioptimalkan untuk CPU |
