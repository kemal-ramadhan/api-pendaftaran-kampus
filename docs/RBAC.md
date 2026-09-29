# Role-Based Access Control (RBAC) & Permission Matrix (docs/RBAC.md)

Dokumen ini mendefinisikan arsitektur RBAC berbasis permission, pemetaan modul portal admin, seeder matriks awal, format header gateway, serta daftar seluruh endpoint admin beserta permission dan role yang memiliki wewenang (berdasarkan `PRD-TAMBAHAN-CR009-010.md` Bagian 1 & 2).

---

## 1. Arsitektur RBAC Berbasis Permission

### 1.1 Prinsip Desain
1. **Pemeriksaan Berbasis Permission, Bukan Hardcoded Role**:
   - Seluruh endpoint administratif (`/admin/*`) pada mikroservis hilir memeriksa permission aksi spesifik melalui middleware `pkg/rbac.RequirePermission("permission.code")`.
   - Nama role tidak lagi di-hardcode di dalam controller layanan downstream.
2. **Klaim Token JWT & Header Gateway**:
   - Access token JWT memuat klaim array `perms` yang berisi daftar unik seluruh kode permission milik user (hasil agregasi seluruh role user).
   - API Gateway membaca klaim `perms` dari token dan memancarkannya ke header internal:
     - `X-User-Permissions: code1,code2,code3`
     - Gateway membersihkan header palsu `X-User-Permissions` yang dikirim dari klien publik.
   - API Gateway tetap menerapkan route guard kasar: mewajibkan login JWT dan memeriksa bahwa pengguna memiliki salah satu role panitia/staf/admin (bukan `calon_mahasiswa` biasa).
3. **Penyaringan Berdasarkan Scope (Data Ownership)**:
   - Pengguna dengan role `prodi` hanya dapat mengakses data yang berasosiasi dengan program studinya (`scope_id = prodi_code`).
   - Pengguna dengan role baru `dekan` memiliki wewenang pada tingkat fakultas (`scope_id = faculty_code`) dan dapat melihat seluruh data program studi di bawah fakultas tersebut.
   - Helper `pkg/rbac.ScopeFilter` mengekstrak konteks `X-User-Roles` dan `X-User-Scopes` untuk disematkan pada klausa query database.
4. **Format Error Standar**:
   - Jika pengguna tidak memiliki permission yang dipersyaratkan, server mengembalikan respons HTTP `403 Forbidden` dengan kode error `PERMISSION_DENIED`:
   ```json
   {
     "success": false,
     "message": "Akses ditolak: izin tidak mencukupi",
     "error_code": "PERMISSION_DENIED",
     "errors": {
       "required_permissions": ["admission.document.verify"]
     },
     "request_id": "req-xxx"
   }
   ```

---

## 2. Matriks Modul & Permission (Seeder Matriks 1.4)

Legenda Hak Akses:
- **K (Kelola)**: Akses penuh baca dan eksekusi aksi manipulasi data/tindakan administratif.
- **B (Baca)**: Akses baca saja (view-only), tanpa wewenang aksi modifikasi/approval.
- **S (Scope)**: Akses baca/kelola terbatas pada unit kerja sendiri (prodi sendiri atau fakultas sendiri).
- **— (None)**: Tidak memiliki akses ke modul tersebut.

| Modul Portal Admin | Permission Kode | superadmin | admin_pmb | keuangan | prodi | dekan | pimpinan | dosen |
|---|---|---|---|---|---|---|---|---|
| **Ringkasan & KPI** | `dashboard.overview.read` | K | K | B (KPI Keuangan) | S (Prodi) | S (Fakultas) | B | — |
| **Verifikasi Berkas** | `admission.document.verify` | K | K | — | S (Baca) | — | — | — |
| **Rekonsiliasi VA** | `finance.reconciliation.read`<br>`finance.payment.manual_confirm` | K | B (Read saja) | K | — | — | B (Read saja) | — |
| **Seleksi & CBT** | `selection.score.read`<br>`selection.interview.score`<br>`selection.recommendation.manage`<br>`selection.approval.decide`<br>`selection.announcement.publish` | K | K (Publish, Wawancara) | — | S (Rekomendasi) | S (Baca) | K (Approve) | S (Wawancara Sesi Sendiri) |
| **Penetapan NIM & PKKMB** | `student.nim.issue`<br>`student.pkkmb.manage`<br>`student.activation.manage` | K | K | B (Status UKT) | S (Baca) | — | B | — |
| **Dashboard Institusi** | `dashboard.institution.read` | K | B | B | S (Tab Prodi) | S (Tab Fakultas) | K (Semua Tab) | — |
| **Laporan & Buku Besar** | `report.export`<br>`report.official.print`<br>`broadcast.send`<br>`finance.ledger.read` | K | K (Kecuali Buku Besar: B) | K (Laporan Kas & Buku Besar) | S (Laporan Prodi) | S (Laporan Fakultas) | B | — |
| **Master Data Prodi** | `organization.program.manage`<br>`finance.fee.manage` | K | B | K (Biaya/UKT saja) | B | B | B | — |
| **Kelola Akun & RBAC** | `auth.user.read`<br>`auth.user.manage`<br>`auth.permission.read`<br>`auth.permission.manage`<br>`auth.role.manage` | K | B (User Read) | — | — | — | B (User Read) | — |

---

## 3. Daftar Lengkap Pemetaan Endpoint Admin ke Permission

Tabel berikut menjadi acuan uji otomatis otorisasi RBAC (test membaca tabel ini untuk memvalidasi penolakan role tanpa permission `403` dan penerimaan role yang berhak):

| No | Service | Method | Path Admin | Permission Dibutuhkan | Scope Diperiksa | Role yang Berhak |
|---|---|---|---|---|---|---|
| 1 | Auth | GET | `/api/v1/auth/me/permissions` | *(Authenticated)* | Tidak | Semua Role Login |
| 2 | Auth | GET | `/api/v1/auth/admin/permissions` | `auth.permission.read` | Tidak | `superadmin` |
| 3 | Auth | GET | `/api/v1/auth/admin/roles/:code/permissions` | `auth.permission.read` | Tidak | `superadmin` |
| 4 | Auth | PUT | `/api/v1/auth/admin/roles/:code/permissions` | `auth.permission.manage` | Tidak | `superadmin` |
| 5 | Auth | GET | `/api/v1/auth/admin/users` | `auth.user.read` | Tidak | `superadmin`, `admin_pmb`, `pimpinan` |
| 6 | Auth | POST | `/api/v1/auth/admin/users` | `auth.user.manage` | Tidak | `superadmin` |
| 7 | Organization | GET | `/api/v1/organization/admin/programs-overview` | `organization.program.read` | Opsional (Prodi/Fakultas) | `superadmin`, `admin_pmb`, `keuangan`, `prodi`, `dekan`, `pimpinan` |
| 8 | Organization | PATCH | `/api/v1/organization/admin/study-programs/:code/admission` | `organization.program.manage` | Tidak | `superadmin` |
| 9 | Organization | POST | `/api/v1/organization/admin/study-programs` | `organization.program.manage` | Tidak | `superadmin` |
| 10 | Organization | PUT | `/api/v1/organization/admin/institution-profile` | `organization.program.manage` | Tidak | `superadmin` |
| 11 | Admission | GET | `/api/v1/admission/admin/header-summary` | `dashboard.overview.read` | Ya (Prodi/Fakultas) | `superadmin`, `admin_pmb`, `keuangan`, `prodi`, `dekan`, `pimpinan` |
| 12 | Admission | GET | `/api/v1/admission/admin/overview` | `dashboard.overview.read` | Ya (Prodi/Fakultas) | `superadmin`, `admin_pmb`, `keuangan`, `prodi`, `dekan`, `pimpinan` |
| 13 | Admission | GET | `/api/v1/admission/admin/document-queue` | `admission.document.verify` | Ya (Prodi jika role prodi) | `superadmin`, `admin_pmb`, `prodi` |
| 14 | Admission | GET | `/api/v1/admission/admin/applications/:id/documents` | `admission.document.verify` | Ya (Prodi jika role prodi) | `superadmin`, `admin_pmb`, `prodi` |
| 15 | Admission | PUT | `/api/v1/admission/admin/applications/:id/documents/:doc_id/verify` | `admission.document.verify` | Ya (Hanya admin/superadmin kelola) | `superadmin`, `admin_pmb` |
| 16 | Admission | GET | `/api/v1/admission/admin/institution-dashboard` | `dashboard.institution.read` | Ya (Level prodi/fakultas/univ) | `superadmin`, `admin_pmb`, `keuangan`, `prodi`, `dekan`, `pimpinan` |
| 17 | Admission | GET | `/api/v1/admission/admin/reports/preview` | `report.export` | Ya (Prodi/Fakultas) | `superadmin`, `admin_pmb`, `prodi`, `dekan`, `pimpinan` |
| 18 | Admission | GET | `/api/v1/admission/admin/reports/applicants/export` | `report.export` | Ya (Prodi/Fakultas) | `superadmin`, `admin_pmb`, `prodi`, `dekan`, `pimpinan` |
| 19 | Admission | GET | `/api/v1/admission/admin/reports/official.pdf` | `report.official.print` | Tidak | `superadmin`, `admin_pmb` |
| 20 | Admission | POST | `/api/v1/admission/admin/path-changes/:id/review` | `admission.path_change.manage` | Tidak | `superadmin`, `admin_pmb` |
| 21 | Admission | POST | `/api/v1/admission/admin/path-changes/direct` | `admission.path_change.manage` | Tidak | `superadmin`, `admin_pmb` |
| 22 | Admission | POST | `/api/v1/admission/admin/withdrawals/:id/review` | `admission.withdrawal.manage` | Tidak | `superadmin`, `admin_pmb` |
| 23 | Admission | POST | `/api/v1/admission/admin/withdrawals/direct` | `admission.withdrawal.manage` | Tidak | `superadmin`, `admin_pmb` |
| 24 | Finance | GET | `/api/v1/finance/admin/reconciliation` | `finance.reconciliation.read` | Tidak | `superadmin`, `admin_pmb`, `keuangan`, `pimpinan` |
| 25 | Finance | GET | `/api/v1/finance/admin/reconciliation/export` | `finance.reconciliation.read` | Tidak | `superadmin`, `keuangan` |
| 26 | Finance | POST | `/api/v1/finance/admin/invoices/:id/manual-confirm` | `finance.payment.manual_confirm` | Tidak | `superadmin`, `keuangan` |
| 27 | Finance | GET | `/api/v1/finance/admin/ledger` | `finance.ledger.read` | Tidak | `superadmin`, `keuangan`, `admin_pmb` |
| 28 | Finance | GET | `/api/v1/finance/admin/fee-schemes` | `finance.fee.manage` | Tidak | `superadmin`, `keuangan`, `admin_pmb` |
| 29 | Finance | PUT | `/api/v1/finance/admin/fee-schemes/:id` | `finance.fee.manage` | Tidak | `superadmin`, `keuangan` |
| 30 | Exam | GET | `/api/v1/exam/admin/sessions` | `selection.score.read` | Tidak | `superadmin`, `admin_pmb`, `dosen` |
| 31 | Exam | POST | `/api/v1/exam/admin/sessions` | `selection.score.read` | Tidak | `superadmin`, `admin_pmb` |
| 32 | Scoring | PUT | `/api/v1/scoring/admin/interview-scores/:application_id` | `selection.interview.score` | Ya (Sesi dosen sendiri jika dosen) | `superadmin`, `admin_pmb`, `dosen` |
| 33 | Selection | GET | `/api/v1/selection/admin/scoreboard` | `selection.score.read` | Ya (Prodi/Fakultas) | `superadmin`, `admin_pmb`, `prodi`, `dekan`, `pimpinan` |
| 34 | Selection | POST | `/api/v1/selection/admin/recommendations` | `selection.recommendation.manage` | Ya (Prodi sendiri) | `superadmin`, `admin_pmb`, `prodi` |
| 35 | Selection | POST | `/api/v1/selection/admin/approval` | `selection.approval.decide` | Tidak | `superadmin`, `pimpinan` |
| 36 | Selection | POST | `/api/v1/selection/admin/announcements/publish` | `selection.announcement.publish` | Tidak | `superadmin`, `admin_pmb` |
| 37 | Student | GET | `/api/v1/student/admin/enrolment` | `student.nim.issue` | Ya (Prodi/Fakultas) | `superadmin`, `admin_pmb`, `keuangan`, `prodi`, `pimpinan` |
| 38 | Student | POST | `/api/v1/student/admin/students/:id/issue-nim` | `student.nim.issue` | Tidak | `superadmin`, `admin_pmb` |
| 39 | Student | POST | `/api/v1/student/admin/nim/batch` | `student.nim.issue` | Tidak | `superadmin`, `admin_pmb` |
| 40 | Student | GET | `/api/v1/student/admin/pkkmb/groups` | `student.pkkmb.manage` | Tidak | `superadmin`, `admin_pmb` |
| 41 | Student | POST | `/api/v1/student/admin/pkkmb/groups` | `student.pkkmb.manage` | Tidak | `superadmin`, `admin_pmb` |
| 42 | Student | POST | `/api/v1/student/admin/pkkmb/auto-assign` | `student.pkkmb.manage` | Tidak | `superadmin`, `admin_pmb` |
| 43 | Student | GET | `/api/v1/student/admin/siakad-export` | `student.activation.manage` | Tidak | `superadmin`, `admin_pmb` |
| 44 | NIM | GET | `/api/v1/nim/admin/formats/active` | `student.nim.issue` | Tidak | `superadmin`, `admin_pmb` |
| 45 | NIM | GET | `/api/v1/nim/admin/formats` | `student.nim.issue` | Tidak | `superadmin` |
| 46 | Notification | GET | `/api/v1/notifications/admin/activity` | `dashboard.overview.read` | Ya | `superadmin`, `admin_pmb`, `keuangan`, `prodi`, `dekan`, `pimpinan` |
| 47 | Notification | GET | `/api/v1/notifications/admin/broadcast/templates` | `broadcast.send` | Tidak | `superadmin`, `admin_pmb` |
| 48 | Notification | POST | `/api/v1/notifications/admin/broadcast/templates` | `broadcast.send` | Tidak | `superadmin`, `admin_pmb` |
| 49 | Notification | POST | `/api/v1/notifications/admin/broadcast/preview` | `broadcast.send` | Tidak | `superadmin`, `admin_pmb` |
| 50 | Notification | POST | `/api/v1/notifications/admin/broadcast` | `broadcast.send` | Tidak | `superadmin`, `admin_pmb` |
| 51 | Notification | GET | `/api/v1/notifications/admin/broadcast` | `broadcast.send` | Tidak | `superadmin`, `admin_pmb` |
