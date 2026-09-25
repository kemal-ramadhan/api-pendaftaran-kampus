# LAPORAN AUDIT MENYELURUH FASE 2 TERHADAP PRD
**Sistem Pendaftaran Mahasiswa Baru (PMB)**  
*Dokumen Sumber Kebenaran: `PRD-LENGKAP.md` (Konsolidasi v3 — 24 September 2026)*  
*Tanggal Audit: 25 September 2026*  
*Status Verifikasi Docker: **BELUM TERVERIFIKASI DI DOCKER** (Docker Desktop Engine host pipe 500)*  
*Status Build & Unit Test: **100% LULUS (33 paket Go lulus build & test)***

---

## 1. Ringkasan Eksekutif Hasil Audit

Audit menyeluruh telah dilakukan terhadap seluruh kode basis Fase 2 (`organization-service`, `identity-service`, `ocr-service`, `notification-service`, `admission-service`, `finance-service`, `exam-service`, `scoring-service`, `selection-service`, `student-service`, `nim-service`) serta komponen integrasi (`api-gateway`, `routes.yaml`, dan `scripts/e2e.sh`).

### Temuan Kunci:
1. **Fondasi Kompilasi & Versi Go Berhasil Diperbaiki**: Versi Go di seluruh `go.mod` telah diseragamkan ke `1.25.0` dan base image Dockerfile ke `golang:1.26-alpine`. Semua dependensi telah di-`tidy` dan `go build ./...`, `go vet ./...`, serta seluruh unit test lulus tanpa error.
2. **Kesenjangan Kritis CR-001 (Duitku Payment Gateway)**: `finance-service` masih menggunakan adapter `MockProvider` saja. File `duitku_provider.go`, method `CheckStatus`, env `DUITKU_*`, dan kolom database pembayaran terkait Duitku belum diimplementasikan.
3. **Celah Keamanan & Batasan Arsitektur**:
   - `identity-service`: tabel `ktp_scans` memiliki kolom `ocr_data_raw TEXT` yang berpotensi menyimpan NIK/data pribadi mentah secara permanen, bertentangan dengan PRD C.4.11 dan UU PDP No. 27/2022.
   - `api-gateway`: rute `/api/v1/ocr` diekspos ke publik/klien, padahal `ocr-service` berstatus internal-only dan hanya boleh diakses melalui `identity-service`.
4. **Endpoint Master Data CRUD Belum Di-mount**: Endpoint CRUD master data admin pada `admission-service` (periode, jalur, kuota, persyaratan dokumen) dan `finance-service` (skema biaya) belum didaftarkan di router.
5. **Artefak Swagger Belum Digenrate**: Seluruh service Fase 2 belum memiliki folder `docs/` berisi file `docs.go`, `swagger.json`, dan `swagger.yaml`.

---

## 2. Tabel Audit Rinci per Service

| Service | Build / Vet / Test | Status Container di Docker | Tabel DB vs PRD | Kesesuaian Endpoint vs PRD | Otorisasi & Gateway Signature | Kepatuhan Aturan Bisnis Kritis | Route Gateway (`routes.yaml`) | Status Dokumentasi |
|---|---|---|---|---|---|---|---|---|
| **Organization** | ✅ LULUS | ⚠️ BELUM TERVERIFIKASI DI DOCKER | Sesuai (`faculties`, `study_programs`) | Sesuai (`/faculties`, `/study-programs`, `/admin/*`, `/internal/*`) | ✅ `gatewayauth` terpasang; Role `superadmin` pada `/admin`; `internalauth` pada `/internal` | Validasi kode unik prodi; scope_id untuk role prodi | Sesuai (`/api/v1/organization/admin`, `/api/v1/organization`) | API.md ✅, Class Diagram ✅, ERD ✅, Postman ✅, Swagger ❌ |
| **Identity** | ✅ LULUS | ⚠️ BELUM TERVERIFIKASI DI DOCKER | ⚠️ Perlu Perbaikan: `ktp_scans` memuat `ocr_data_raw TEXT` (pelanggaran UU PDP) | Sesuai (`/me/*`, `/admin/*`, `/internal/*`, `/me/ktp-scan/*`) | ✅ `gatewayauth` terpasang; Role `admin_pmb`/`superadmin` pada `/admin`; IDOR terlindungi (`X-User-ID`) | NIK tersimpan terenkripsi AES-256-GCM + hash SHA-256; NIK dimasking untuk non-pemilik/admin; Presigned URL foto | Sesuai (`/api/v1/identity/admin`, `/api/v1/identity`) | API.md ✅, Class Diagram ✅, ERD ✅, Postman ✅, Swagger ❌ |
| **OCR** *(Python)* | ✅ LULUS (Script/Fixture) | ⚠️ BELUM TERVERIFIKASI DI DOCKER | Sesuai (Tanpa DB permanen, Redis TTL 15m) | Sesuai (`/internal/ocr/ktp`, `/internal/ocr/jobs/:id`, `/health`, `/health/ready`) | ✅ `internalauth` terpasang (`X-Internal-Token`) | Non-generatif (CLAHE, unsharp, bilateral); deteksi kualitas & `retake_required`; validasi NIK wilayah; log bebas NIK | ⚠️ SALAH: `/api/v1/ocr` diekspos di Gateway (seharusnya internal only) | API.md ✅, Benchmark ✅ (`docs/ocr-benchmark.md`), Swagger (Auto FastAPI), Postman ✅ |
| **Notification** | ✅ LULUS | ⚠️ BELUM TERVERIFIKASI DI DOCKER | Sesuai (`notification_logs`) | ⚠️ Kurang `/health/ready` (hanya `/health`); internal notification sesuai | ✅ `internalauth` terpasang (`X-Internal-Token`) | Antrean asinkron Redis + worker; retry 3x backoff; Mailpit support; template Bahasa Indonesia lengkap | Sesuai (Internal only, tidak diekspos di Gateway) | API.md ✅, Class Diagram ✅, ERD ✅, Postman ✅, Swagger ❌ |
| **Admission** | ✅ LULUS | ⚠️ BELUM TERVERIFIKASI DI DOCKER | Sesuai (`periods`, `paths`, `quotas`, `requirements`, `applications`, `documents`, `logs`) | ⚠️ Kurang: CRUD master data admin (`/admin/periods`, `/admin/paths`, `/admin/quotas`, `/admin/requirements`) belum ada di router | ✅ `gatewayauth` terpasang; Role guard lengkap; IDOR terlindungi | State machine 8 tahap divalidasi ketat; batas periode registrasi; 1 pendaftaran aktif per gelombang; Magic bytes MIME check; Presigned URL 5 menit | Sesuai (`/api/v1/admission/admin`, `/api/v1/admission`) | API.md ✅, Class Diagram ✅, ERD ✅, Postman ✅, Swagger ❌ |
| **Finance** | ✅ LULUS | ⚠️ BELUM TERVERIFIKASI DI DOCKER | ⚠️ Perlu Perbaikan: kolom Duitku (`payment_url`, `payment_code`, dll.) belum ada; `study_program_id UUID` vs PRD `study_program_code` | ⚠️ Kurang: CRUD `/admin/fee-schemes` belum ada di router; Duitku webhook path di gateway tidak seragam | ✅ `gatewayauth` terpasang; Role `keuangan`/`superadmin` pada `/admin` | ⚠️ KRITIS: `DuitkuProvider` (CR-001) belum diimplementasikan (hanya mock); Uang integer; Cicilan exact sum; Kwitansi PDF; Idempotensi webhook | ⚠️ Perlu Penyesuaian: webhook publik di `/api/v1/webhooks/payments` vs PRD `/api/v1/finance/webhooks/payments/duitku` | API.md ✅, Class Diagram ✅, ERD ✅, Postman ✅, Swagger ❌ |
| **Exam** | ✅ LULUS | ⚠️ BELUM TERVERIFIKASI DI DOCKER | Sesuai (`exam_sessions`, `session_examiners`, `participants`, `exam_cards`) | ⚠️ Kurang `/health/ready` (hanya `/health`); endpoint jadwal, kartu, dan presensi sesuai | ✅ `gatewayauth` terpasang; Role `dosen`, `admin_pmb`, `superadmin` | Hanya pendaftaran `verified` yang dijadwalkan; Kartu PDF lengkap QR code; Proteksi kapasitas sesi | Sesuai (`/api/v1/exam/admin`, `/api/v1/exam/examiner`, `/api/v1/exam/me`) | API.md ✅, Class Diagram ✅, ERD ✅, Postman ✅, Swagger ❌ |
| **Scoring** | ✅ LULUS | ⚠️ BELUM TERVERIFIKASI DI DOCKER | Sesuai (`score_components`, `scores`, `score_summaries`) | ⚠️ Kurang `/health/ready` (hanya `/health`); endpoint komponen, input nilai, rekap ranking sesuai | ✅ `gatewayauth` terpasang; Role guard dosen (hanya sesi terkait) & prodi (scope sendiri) | Bobot komponen total 100%; Nilai terkunci pasca-submit rekomendasi; Perhitungan tertimbang otomatis | Sesuai (`/api/v1/scoring/admin`, `/api/v1/scoring/examiner`, `/api/v1/scoring/program`) | API.md ✅, Class Diagram ✅, ERD ✅, Postman ✅, Swagger ❌ |
| **Selection** | ✅ LULUS | ⚠️ BELUM TERVERIFIKASI DI DOCKER | Sesuai (`recommendations`, `recommendation_items`, `announcements`, `results`) | ⚠️ Kurang `/health/ready` (hanya `/health`); endpoint rekomendasi, approval, publikasi, hasil sesuai | ✅ `gatewayauth` terpasang; Role prodi, pimpinan, admin_pmb | Rekomendasi dibatasi kuota prodi; Hasil tersembunyi sebelum `published_at`; Publikasi otomatis trigger tagihan daftar ulang | Sesuai (`/api/v1/selection/program`, `/api/v1/selection/approval`, `/api/v1/selection/admin`, `/api/v1/selection/me`) | API.md ✅, Class Diagram ✅, ERD ✅, Postman ✅, Swagger ❌ |
| **Student** | ✅ LULUS | ⚠️ BELUM TERVERIFIKASI DI DOCKER | Sesuai (`reregistrations`, `students`, `outbox_events`) | Sesuai (`/me/reregistration`, `/admin/students`, export xlsx, `/internal/*`) | ✅ `gatewayauth` terpasang; Role guard lengkap; `/health` & `/health/ready` ada | Syarat daftar ulang (lulus seleksi + lunas / termin 1 + data tambahan); Outbox event NIM | Sesuai (`/api/v1/student/admin`, `/api/v1/student/me`) | API.md ✅, Class Diagram ✅, ERD ✅, Postman ✅, Swagger ❌ |
| **NIM** | ✅ LULUS | ⚠️ BELUM TERVERIFIKASI DI DOCKER | Sesuai (`nim_formats`, `nim_sequences`, `nim_issuances`, `outbox_events`) | Sesuai (`/admin/issuances`, `/admin/formats`, `/internal/nim/issue`) | ✅ `gatewayauth` terpasang; Role guard superadmin/admin_pmb; `/health` & `/health/ready` ada | `SELECT ... FOR UPDATE` mencegah NIM ganda; Idempoten per `application_id`; Outbox event role mahasiswa | Sesuai (`/api/v1/nim/admin/formats`, `/api/v1/nim/admin/issuances`, `/api/v1/nim/admin`) | API.md ✅, Class Diagram ✅, ERD ✅, Postman ✅, Swagger ❌ |

---

## 3. Tabel Keterlacakan Use Case (UC-01 s/d UC-33 + UC-06a)

| No. UC | Nama Use Case | Service Utama | Status PRD | Status Verifikasi Saat Ini | Catatan Audit |
|---|---|---|---|---|---|
| **UC-01** | Login | Auth | Fase 1 | ✅ **Selesai & teruji** | Lulus E2E 43/43 dari DB kosong |
| **UC-02** | Registrasi Akun | Auth | Fase 1 | ✅ **Selesai & teruji** | Lulus E2E 43/43 dari DB kosong |
| **UC-03** | Verifikasi Email / OTP | Auth | Fase 1 | ✅ **Selesai & teruji** | Lulus E2E 43/43 dari DB kosong |
| **UC-04** | Pilih Gelombang & Jalur | Admission | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Kode & unit test lulus; perlu diverifikasi E2E Docker |
| **UC-05** | Pilih Program Studi | Admission (+ Organization) | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Validasi kode prodi aktif berjalan di unit test |
| **UC-06** | Isi Biodata | Identity | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Profil, alamat, orang tua, riwayat pendidikan lulus test |
| **UC-06a** | Scan KTP (OCR + perbaikan citra) | Identity + OCR | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Benchmark sintetis memenuhi target; perlu verifikasi container |
| **UC-07** | Upload Dokumen Persyaratan | Admission | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Magic bytes MIME validation & MinIO storage teruji unit test |
| **UC-08** | Bayar Biaya Pendaftaran | Finance | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | MockProvider teruji; Duitku adapter belum diimplementasikan |
| **UC-09** | Cetak Kartu Ujian | Exam | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Generator PDF & QR code teruji unit test |
| **UC-10** | Lihat Pengumuman Hasil | Selection | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Filter `published_at` teruji unit test |
| **UC-11** | Bayar Daftar Ulang | Finance | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Skema lunas & cicilan termin teruji unit test |
| **UC-12** | Daftar Ulang | Student | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Syarat lunas/termin 1 & kelengkapan data teruji unit test |
| **UC-13** | Proses Pembayaran | Finance | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | MockProvider teruji; Duitku adapter belum diimplementasikan |
| **UC-14** | Upload Bukti Transfer | Finance | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Status `awaiting_verification` teruji unit test |
| **UC-15** | Ajukan Cicilan | Finance | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Pembagian termin integer exact sum teruji unit test |
| **UC-16** | Generate NIM | NIM | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Concurrency safety `SELECT FOR UPDATE` teruji unit test |
| **UC-17** | Kirim Notifikasi | Notification | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Antrean Redis & retry backoff teruji unit test |
| **UC-18** | Kelola Data Pendaftar | Admission | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Filter status, prodi, jalur teruji unit test |
| **UC-19** | Verifikasi Dokumen | Admission | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Alur verifikasi / revisi teruji unit test |
| **UC-20** | Kelola Jadwal Seleksi | Exam | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | CRUD sesi ujian & kapasitas teruji unit test |
| **UC-21** | Assign Dosen Penguji | Exam | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Pengikatan dosen ke sesi seleksi teruji unit test |
| **UC-22** | Publikasikan Pengumuman | Selection | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Trigger tagihan daftar ulang teruji unit test |
| **UC-23** | Verifikasi Pembayaran | Finance | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Approve/reject manual transfer teruji unit test |
| **UC-24** | Setujui Pengajuan Cicilan | Finance | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Review pengajuan cicilan teruji unit test |
| **UC-25** | Lihat Laporan Keuangan | Finance | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Ringkasan pendapatan & tagihan teruji unit test |
| **UC-26** | Export Laporan (Excel) | Finance, Scoring, Admission, Student | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Format XLSX generated via Excelize teruji unit test |
| **UC-27** | Input Nilai Seleksi | Scoring | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Dosen hanya menginput nilai pesertanya teruji unit test |
| **UC-28** | Lihat Rekap & Ranking | Scoring | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Perhitungan nilai akhir tertimbang teruji unit test |
| **UC-29** | Rekomendasi Kelulusan | Selection | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Validasi tidak melebihi kuota teruji unit test |
| **UC-30** | Setujui Kelulusan | Selection | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Pimpinan approve/return teruji unit test |
| **UC-31** | Lihat Dashboard Statistik | Admission, Finance, Student, NIM | Fase 2 | ⚠️ **Ada tapi belum teruji di Docker** | Agregasi data eksekutif pimpinan teruji unit test |
| **UC-32** | Kelola Master Data | Org, Admission, Finance | Fase 2 | ⚠️ **Kurang** | Master prodi ada; Master gelombang, jalur, kuota, dokumen, skema biaya belum didaftarkan di router |
| **UC-33** | Kelola User & Role | Auth | Fase 1 | ✅ **Selesai & teruji** | Lulus E2E 43/43 dari DB kosong |

---

## 4. Daftar Temuan Terurut Prioritas

### A. Prioritas KRITIS (Keamanan, Integritas Data, Alur Putus)
1. **[KRITIS-1] CR-001: Adapter Duitku POP (`createInvoice` & callback) Belum Diimplementasikan**
   - **Komponen**: `services/finance-service` (`internal/provider/`, `internal/config/`, `migrations/`).
   - **Deskripsi**: Interface `PaymentProvider` saat ini masih berorientasi mock (`CreateVA`). Belum ada implementasi `DuitkuProvider`, belum ada method `CheckStatus` untuk rekonsiliasi berkala, belum ada env `DUITKU_*`, dan tabel `payments` belum memiliki kolom `payment_url`, `payment_code`, `publisher_order_id`, `settlement_date`.
   - **Rekomendasi**: Implementasikan `DuitkuProvider` dengan HMAC-SHA256, sesuaikan interface `PaymentProvider`, tambahkan migration untuk field Duitku, dan implementasikan unit test dengan `httptest.Server`.

2. **[KRITIS-2] Potensi Pelanggaran UU PDP pada Kolom `ktp_scans.ocr_data_raw`**
   - **Komponen**: `services/identity-service` (`migrations/000001_create_identity_tables.up.sql`).
   - **Deskripsi**: Tabel `ktp_scans` menyimpan kolom `ocr_data_raw TEXT` yang memuat teks mentah hasil pembacaan OCR (berisi NIK dan biodata lengkap). PRD C.4.11 dan UU No. 27/2022 secara tegas melarang penyimpanan teks mentah OCR secara permanen.
   - **Rekomendasi**: Buang kolom `ocr_data_raw` dari tabel `ktp_scans`. Cukup simpan metadata (`status`, `quality_score`, `enhanced`, `field_confidences` tanpa NIK, `fields_edited_by_user`).

3. **[KRITIS-3] Pemaparan Endpoint Internal OCR Service di API Gateway**
   - **Komponen**: `services/api-gateway` (`config/routes.yaml`).
   - **Deskripsi**: Terdapat rute `/api/v1/ocr` yang me-rewrite request ke `/internal/ocr` pada `ocr-service`. Berdasarkan PRD C.4.11 dan C.2, `ocr-service` berstatus internal-only dan tidak boleh diekspos langsung di API Gateway. Interaksi klien wajib melalui `identity-service` (`/api/v1/identity/me/ktp-scan`).
   - **Rekomendasi**: Hapus rute `/api/v1/ocr` dari `routes.yaml`.

---

### B. Prioritas TINGGI (Fitur Utama Kurang / Ketidaksesuaian Kontrak PRD)
1. **[TINGGI-1] Endpoint Master Data Admin pada `admission-service` Belum Di-mount di Router**
   - **Komponen**: `services/admission-service` (`internal/route/router.go`).
   - **Deskripsi**: PRD C.4.4 mensyaratkan endpoint CRUD `/admin/periods`, `/admin/paths`, `/admin/quotas`, `/admin/requirements` untuk `superadmin`, namun rute tersebut belum ada di `SetupRouter`.
   - **Rekomendasi**: Implementasikan controller handler dan daftarkan route CRUD master data tersebut di bawah grup `/admission/admin` dengan guard role `superadmin`.

2. **[TINGGI-2] Endpoint Master Data Skema Biaya pada `finance-service` Belum Di-mount di Router**
   - **Komponen**: `services/finance-service` (`internal/route/router.go`).
   - **Deskripsi**: PRD C.4.5 mensyaratkan CRUD `/admin/fee-schemes` untuk `superadmin` dan `keuangan`, namun handler dan rutenya belum didaftarkan di router.
   - **Rekomendasi**: Tambahkan controller dan rute CRUD `/admin/fee-schemes`.

3. **[TINGGI-3] Inkonsistensi Kolom Prodi pada `fee_schemes`**
   - **Komponen**: `services/finance-service` (`migrations/000001_create_finance_tables.up.sql`).
   - **Deskripsi**: Tabel `fee_schemes` menggunakan `study_program_id UUID`, sedangkan Organization Service menggunakan kode string (`code`, misal `TI`). Di PRD C.4.5 kolomnya adalah `study_program_code VARCHAR(20) NULL`.
   - **Rekomendasi**: Ubah kolom menjadi `study_program_code VARCHAR(20)` agar konsisten dengan `Organization Service`.

---

### C. Prioritas SEDANG (Standar Teknis & Kerapian Arsitektur)
1. **[SEDANG-1] Inkonsistensi Endpoint Readiness `/health/ready`**
   - **Komponen**: `notification-service`, `exam-service`, `scoring-service`, `selection-service`.
   - **Deskripsi**: Layanan-layanan tersebut hanya menyediakan `/health` tanpa `/health/ready`, bertentangan dengan standar PRD Bagian B dan E.2 yang mensyaratkan liveness (`/health`) dan readiness (`/health/ready`).
   - **Rekomendasi**: Tambahkan handler dan endpoint `/health/ready` pada keempat service tersebut.

2. **[SEDANG-2] Standardisasi Prefix Rute Gateway untuk Pembayaran**
   - **Komponen**: `services/api-gateway` (`config/routes.yaml`).
   - **Deskripsi**: Rute webhook callback berada di `/api/v1/webhooks/payments`, sedangkan PRD D.3 menyebutkan `/api/v1/finance/webhooks/payments/duitku`. Demikian pula `/api/v1/invoices` dan `/api/v1/receipts` sebaiknya memiliki alias `/api/v1/finance/invoices` dan `/api/v1/finance/receipts`.
   - **Rekomendasi**: Tambahkan prefix yang seragam di `routes.yaml` tanpa merusak kompatibilitas ke belakang.

---

### D. Prioritas RENDAH (Dokumentasi & Swaggo)
1. **[RENDAH-1] Folder Artefak Swagger Belum Digenrate di Service Fase 2**
   - **Komponen**: Seluruh service Fase 2 (10 service Go).
   - **Deskripsi**: Folder `docs/` berisi artefak Swaggo (`docs.go`, `swagger.json`, `swagger.yaml`) saat ini baru tersedia di `auth-service`.
   - **Rekomendasi**: Jalankan `swag init` di setiap service saat pengerjaan tahapan masing-masing.

---

## 5. Status Script Pengujian E2E Saat Ini

File `scripts/e2e.sh` telah dirancang mencakup 16 Bagian Pengujian (118 langkah asersi):
- Bagian 1: Alur Utama Calon Mahasiswa (Register s/d Logout)
- Bagian 2: Skenario Negatif & Keamanan (Bypass Gateway, Spoofing Header, Lockout, Token Revocation)
- Bagian 3: Alur RBAC Administrator & Internal
- Bagian 4: Tahap 1 — Organization Service & Scope Validation
- Bagian 5: Tahap 2 — Identity Service & OCR KTP
- Bagian 7: Tahap 3 — Notification & Admission Service (Part 1)
- Bagian 8: Tahap 4 — Finance Service (Part 1 - Tagihan, VA, Manual Transfer, Kwitansi)
- Bagian 9: Tahap 5 — Admission Service (Part 2 - Berkas, Magic Bytes, Verifikasi, Revisi, Export XLSX)
- Bagian 10: Tahap 6 — Exam Service (Sesi Ujian, Dosen Penguji, Kartu Ujian PDF, Presensi)
- Bagian 11: Tahap 7 — Scoring Service (Komponen Nilai, Input Nilai Dosen, Rekap Ranking)
- Bagian 12: Tahap 8 — Selection Service (Rekomendasi Kuota, Approval Pimpinan, Pengumuman)
- Bagian 13: Tahap 9 — Finance Service (Part 2 - Cicilan, Jatuh Tempo, Laporan Eksekutif)
- Bagian 14: Tahap 10 — Student Service (Daftar Ulang, Data Tambahan, Ekspor XLSX)
- Bagian 15: Tahap 11 — NIM Service & Integrasi Multi-Layanan (Penerbitan NIM, Role Mahasiswa)
- Bagian 16: Tahap 12 — Integrasi Akhir & Dashboard Statistik Pimpinan (UC-31)

### Status Eksekusi:
- **BELUM TERVERIFIKASI DI DOCKER**: Karena lingkungan Docker Desktop host mengembalikan error internal pipe `http:////./pipe/dockerDesktopLinuxEngine/v1.56/info`, script `e2e.sh` belum dijalankan di container Docker yang sehat pada checkpoint ini.
- Seluruh 33 paket Go di 12 module Go telah diuji secara lokal via `go test` dan menghasilkan status **100% PASS**.

---

## 6. Rencana Kerja Perbaikan (Tahap 1 s/d 11)

Perbaikan akan dieksekusi secara terisolasi per tahap sesuai PRD Bagian E.3:
1. **Tahap 1 (Organization)**: Validasi scope prodi di Auth, verifikasi signature gateway.
2. **Tahap 2 (Identity + OCR)**: Hapus `ocr_data_raw` dari DB, tutup rute `/api/v1/ocr` di gateway, validasi enkripsi NIK dan benchmark OCR sintetis.
3. **Tahap 3 (Notification + Admission 1)**: Tambah `/health/ready`, implementasikan CRUD master data period & path, validasi state machine pendaftaran.
4. **Tahap 4 (Finance 1 + CR-001 Duitku)**: Implementasi penuh `DuitkuProvider` (createInvoice, verifySignature HMAC, ParseWebhook, CheckStatus), migration kolom Duitku, CRUD fee schemes, unit test mock server.
5. **Tahap 5 (Admission 2)**: Presigned URL dokumen MinIO, validasi magic bytes file, revisi dokumen, export XLSX.
6. **Tahap 6 (Exam)**: Tambah `/health/ready`, verifikasi penjadwalan hanya bagi status verified, kartu ujian PDF.
7. **Tahap 7 (Scoring)**: Tambah `/health/ready`, pembobotan 100%, penguncian nilai, rekap ranking.
8. **Tahap 8 (Selection)**: Tambah `/health/ready`, validasi kuota prodi, persetujuan pimpinan, trigger otomatis tagihan daftar ulang.
9. **Tahap 9 (Finance 2)**: Alur cicilan exact sum termin, rekonsiliasi Duitku, laporan keuangan ringkasan.
10. **Tahap 10 (Student)**: Verifikasi kelayakan daftar ulang, outbox event penerbitan NIM.
11. **Tahap 11 (NIM & Integrasi Penuh)**: Konkurensi nomor urut NIM (`SELECT FOR UPDATE`), event penambahan role mahasiswa, eksekusi e2e penuh seluruh alur.
