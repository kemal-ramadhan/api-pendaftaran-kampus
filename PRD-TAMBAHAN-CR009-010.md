# PRD Tambahan — CR-009 & CR-010

| | |
|---|---|
| **Produk** | Sistem PMB (backend) — repo `diaricakarsa/pmb-backend` |
| **Isi** | CR-009 Portal Admin (8 modul) & RBAC berbasis permission · CR-010 Penyesuaian desain Portal Calon Mahasiswa v2 |
| **Sumber** | Desain Figma `PMB` v2 (24 halaman: 01–12 portal calon mahasiswa, 13–24 portal admin) diekspor 29 Sep 2026; arahan lead 29 Sep 2026: *"di admin pakai sistem RBAC dan ROLE, yang bisa akses admin bukan hanya admin super tapi keuangan, prodi juga dengan akses terbatas"* |
| **Versi** | v1 — 29 September 2026 |
| **Hubungan dengan dokumen lain** | Dokumen **terpisah**. Dasar aturan umum tetap `PRD-LENGKAP.md` (v3.3). Dokumen ini dikerjakan **setelah** CR-005 dan CR-006..008 (`PRD-TAMBAHAN-CR006-008.md`). Bila bertentangan dengan dokumen sebelumnya, **dokumen ini yang berlaku** untuk bagian yang disebut. |

---

## 0. Instruksi untuk AI Agent

1. Agent tidak punya akses Figma. **Spesifikasi di dokumen ini adalah acuan.** Label UI (teks Bahasa Indonesia) tetap dikirim backend lewat field `*_label` seperti aturan CR-005 (D.14).
2. Setiap endpoint tampilan baru wajib punya JSON Schema di `docs/contracts/` dan diuji (unit/integration + e2e), sesuai CR-005 D.14.
3. Jangan mengubah perilaku yang sudah lulus e2e kecuali disebut eksplisit di sini. Perubahan yang **mengganti** perilaku lama ditandai **[UBAH]**.
4. Semua keputusan default di Bagian 4 dicatat di `docs/DECISIONS.md` dengan tanda "default, menunggu konfirmasi lead".
5. CR-009 dan CR-010 dicatat di `docs/CHANGE_REQUESTS.md` (status `open` → `done`). Tambahkan baris baru ke `docs/FE-BE-GAP.md` untuk setiap halaman 05–24 (status `Open` → `Done`).

---

## 1. CR-009 — RBAC berbasis permission

### 1.1 Masalah
Saat ini akses dicek per **role** (mis. `keuangan`, `prodi`). Lead meminta portal admin bisa dibuka banyak role dengan **akses terbatas per modul dan per aksi**, dan frontend perlu tahu menu/aksi mana yang boleh ditampilkan.

### 1.2 Desain
- **Permission** adalah kode aksi, format `<modul>.<objek>.<aksi>`, contoh `finance.payment.manual_confirm`.
- Auth Service menambah tabel `permissions` (`code`, `module`, `description`) dan `role_permissions` (`role_id`, `permission_code`). Diisi seeder dari matriks 1.4; superadmin dapat mengubah pemetaan lewat API (UI menyusul).
- **Role baru `dekan`** dengan scope `faculty` (kode fakultas dari Organization). Validasi scope sama seperti `prodi`.
- **Token akses** menambah klaim `perms` (daftar kode permission hasil gabungan semua role user). Gateway meneruskannya ke service sebagai header `X-User-Permissions` (dipisah koma), sama seperti `X-User-Roles`, dan tetap membersihkan header palsu dari klien.
- Shared package baru `pkg/rbac`: middleware `RequirePermission(code...)` dan helper scope (`ScopeFilter` untuk `prodi` → prodi, `dekan` → semua prodi di fakultasnya, lewat internal Organization dengan cache).
- **[UBAH]** Endpoint admin di semua service dipindah dari cek role ke cek permission. Route guard di gateway tetap **kasar** (wajib login + salah satu role staf), pemeriksaan permission dan scope **di service**.
- Perubahan pemetaan role→permission berlaku saat token berikutnya diterbitkan (maks. umur access token 15 menit). Endpoint `POST /auth/admin/roles/:code/permissions` mencatat audit log.

### 1.3 Endpoint
| Method | Path (gateway `/api/v1/auth`) | Keterangan |
|---|---|---|
| GET | `/me/permissions` | `{ roles, scopes, permissions: [...], modules: [{key, label, sublabel, allowed, read_only}] }` — frontend memakai `modules` untuk menampilkan menu admin (8 modul di 1.4) dan tombol "Portal Calon Mahasiswa" |
| GET | `/admin/permissions` | Daftar semua permission (superadmin) |
| GET/PUT | `/admin/roles/:code/permissions` | Lihat / ganti pemetaan (superadmin; tidak boleh mengurangi permission superadmin) |

### 1.4 Matriks awal (seeder)
Legenda: **K** kelola (baca + aksi), **B** baca saja, **S** baca dengan scope (prodi/fakultas sendiri), **—** tidak ada akses.

| Modul (desain) | Permission utama | superadmin | admin_pmb | keuangan | prodi | dekan | pimpinan | dosen |
|---|---|---|---|---|---|---|---|---|
| Ringkasan & KPI | `dashboard.overview.read` | K | K | B (KPI keuangan saja) | S | S | B | — |
| Verifikasi Berkas | `admission.document.verify` | K | K | — | S (baca) | — | — | — |
| Rekonsiliasi VA | `finance.reconciliation.read`, `finance.payment.manual_confirm` | K | B | K | — | — | B | — |
| Seleksi & CBT | `selection.score.read`, `selection.interview.score`, `selection.recommendation.manage`, `selection.approval.decide`, `selection.announcement.publish` | K | K (publish, input wawancara) | — | S (rekomendasi) | S (baca) | K (approve) | S (input wawancara sesi sendiri) |
| Penetapan NIM & PKKMB | `student.nim.issue`, `student.pkkmb.manage`, `student.activation.manage` | K | K | B (status UKT) | S (baca) | — | B | — |
| Dashboard Institusi | `dashboard.institution.read` | K | B | B | S (tab Program Studi) | S (tab Fakultas) | K (semua tab) | — |
| Laporan, Broadcast & Buku Besar | `report.export`, `report.official.print`, `broadcast.send`, `finance.ledger.read` | K | K (kecuali buku besar: B) | K (laporan kas & buku besar) | S (laporan prodinya) | S | B | — |
| Master Data Prodi | `organization.program.manage`, `finance.fee.manage` | K | B | K (biaya/UKT saja) | B | B | B | — |

Detail permission lengkap disusun agent di `docs/RBAC.md` dari tabel ini (satu baris per endpoint admin → permission yang dibutuhkan).

---

## 2. CR-009 — Portal Admin (halaman 13–24)

Header semua halaman admin: nama institusi, lencana "Panitia PMB", **pemilih gelombang** (semua endpoint admin menerima `period_id`; default gelombang aktif), tautan "Portal Calon Mahasiswa", identitas user. Tombol **"Sinkronkan Data"** → `POST /admin/cache/refresh` di service terkait (menghapus cache agregat; rate limit 1×/10 detik per user). Label jumlah pendaftar di header dari `GET /admission/admin/header-summary?period_id=`.

### 2.1 Ringkasan Eksekutif & KPI (13)
`GET /admission/admin/overview?period_id=` (agregasi Admission, Finance, Exam, Selection, Student, Organization; di-cache 60 detik; hormati scope):
- Kartu: **Total Pendaftar** (akun calon mahasiswa terdaftar), **Antrean Verifikasi Berkas** (menunggu validasi), **Penerimaan Pembayaran** (Rp, formulir + UKT terverifikasi), **Dinyatakan Lulus Seleksi** (+ tingkat kelulusan %).
- **Proporsi peminat per prodi** (donat): `[{program_code, label, count}]`.
- **Funnel konversi** 6 tahap: Pendaftar Akun → Formulir Lunas → Berkas Terverifikasi → Peserta CBT → Lulus Seleksi → NIM Diterbitkan, plus konversi akhir %.
- **Distribusi peminat & keterisian kuota**: per prodi `terisi / kuota kursi (%)` (terisi = lulus diterima).
- **Log aktivitas panitia**: 20 aktivitas terbaru (lihat 2.9).
- Role `keuangan` hanya menerima kartu/komponen keuangan (field lain `null`).

### 2.2 Verifikasi Berkas (14, 15)
- **[UBAH] Persyaratan dokumen default** menjadi 5 untuk semua jalur: `IJAZAH_SKL` (Ijazah/SKL, pindaian legalisir), `KTP_KIA_KP` (KTP/KIA/Kartu Pelajar), `KK`, `PAS_FOTO` (4×6), **`SURAT_SEHAT`** (surat keterangan sehat dari puskesmas/RS/klinik) — baru. Persyaratan tambahan per jalur (rapor, sertifikat, transkrip) tetap bisa ditambahkan di master.
- `GET /admission/admin/document-queue?period_id=&status=&q=&page=`: filter status `all|waiting|verified|revision|not_uploaded` dengan **jumlah per status** di `meta.counts`; pencarian nama, nomor registrasi, atau **NIK** (Admission memanggil Identity internal `POST /internal/persons/search-nik` dengan hash NIK; NIK tidak pernah dikirim mentah di log). Kolom: nomor registrasi, NIK termask, nama, asal sekolah, prodi pilihan, kelengkapan `x/5 sah`, status berkas + label.
- `GET /admission/admin/applications/:id/documents` untuk modal: data ringkas pendaftar, daftar 5 dokumen berurutan (tab), status per dokumen, `preview_url` (presigned 5 menit), ringkasan `x/5 disetujui`, `next_application_id` (tombol "Berikutnya" ke antrean berikutnya dengan filter yang sama).
- Aksi per dokumen: **Terima Berkas** / **Minta Revisi** (catatan wajib) → endpoint verifikasi yang sudah ada, per dokumen.

### 2.3 Rekonsiliasi Keuangan & Virtual Account (16)
`GET /finance/admin/reconciliation?period_id=`:
- Kartu: **Total formulir terbayar** `x / y` (+ %), **Perlu konfirmasi admin** (jumlah transaksi transfer manual / VA yang perlu ditinjau), **Pelunasan UKT semester 1** (`x dari y` mhs + total dana), **Kanal pembayaran** (persentase per bank/kanal, mis. BSI & Mandiri; dari kode metode Duitku).
- Grafik: **proporsi dana** formulir vs UKT; **distribusi per kanal**.
- Daftar tagihan: nomor invoice, nama & prodi, jenis tagihan, nominal, kanal bayar, status + label, aksi. Paginasi, filter status/kanal/jenis, pencarian.
- `GET /finance/admin/reconciliation/export?format=csv|xlsx`.
- **Konfirmasi Lunas manual** — `POST /finance/admin/invoices/:id/manual-confirm`, hanya permission `finance.payment.manual_confirm` (default: `keuangan`, `superadmin`). **Wajib**: nominal (harus sama dengan sisa tagihan/termin), tanggal & jam transaksi, kanal/bank, **nomor referensi mutasi bank** (unik — ditolak bila pernah dipakai), bukti (PDF/JPG/PNG ≤ 5 MB) dan alasan. Efek: membuat pembayaran `method = manual_confirmation`, membatalkan pembayaran Duitku `pending` pada tagihan itu, kwitansi terbit, audit log, notifikasi ke superadmin. Opsi `MANUAL_CONFIRM_SECOND_APPROVAL=false` (bila `true`, butuh persetujuan pengguna keuangan kedua). Konfirmasi manual **tidak boleh** dilakukan pada tagihan yang sudah lunas (`409 INVOICE_ALREADY_PAID`).

### 2.4 Seleksi Akademik, Nilai CBT & Wawancara (17)
`GET /selection/admin/scoreboard?period_id=&decision=&q=`:
- **Nilai ambang batas minimal** (default 70.0) → `selection_settings.passing_threshold` per gelombang (dapat diubah `selection.approval.decide`/superadmin). **[UBAH]** menggantikan `passing_score` per ujian sebagai acuan keputusan; `passing_score` CBT (CR-005) tetap ditampilkan ke peserta dan default-nya mengikuti ambang ini.
- Kolom: nomor peserta (= nomor registrasi), nama & prodi, **skor CBT**, **rata-rata rapor**, **nilai wawancara**, **status keputusan** (`passed` "Lulus Seleksi", `plenary` "Evaluasi Pleno", `failed` "Tidak Lulus"), aksi **Nilai Wawancara**.
- Filter: Semua, Lulus Seleksi, Evaluasi Pleno, Tidak Lulus (jumlah per filter).
- Keputusan awal otomatis (saran): nilai akhir ≥ ambang → `passed`, kurang → `plenary` (bukan langsung `failed`); keputusan final tetap alur rekomendasi prodi → persetujuan pimpinan → publikasi (C.4.8). Status `plenary` = menunggu rapat pleno.
- **Input nilai wawancara**: `PUT /scoring/admin/interview-scores/:application_id` (permission `selection.interview.score`; dosen hanya peserta sesinya, admin_pmb semua).
- **Rata-rata rapor**: field baru di Identity `education_histories.report_average` (0–100, 2 desimal; diisi calon mahasiswa di biodata bagian Riwayat Sekolah, atau admin). Scoring memakai nilai ini untuk komponen `report_card`.
- **Nomor SK Kelulusan**: saat publikasi, `announcements.decree_number` (wajib, mis. SK Rektor) dan `decree_date`.
- **Pilihan kedua** (dari desain 07: *"Pilihan alternatif diproses apabila kuota pilihan utama telah penuh"*): bila pendaftar memenuhi ambang tetapi kuota prodi pilihan utama penuh, rekap prodi **pilihan kedua** menampilkan pendaftar itu dengan tanda "Pilihan Kedua"; bila diterima, `results.accepted_program` = prodi kedua.

### 2.5 Penerbitan NIM, Gugus PKKMB & Registrasi (18)
- `GET /student/admin/enrolment?period_id=&filter=&q=`: filter **Semua**, **Sudah Ada NIM**, **Siap Terbit NIM** (lulus + UKT lunas/termin 1 terbayar + daftar ulang selesai), **Menunggu UKT**; kolom: nomor registrasi, nama & prodi, status seleksi, status UKT, NIM, **gugus PKKMB**, aksi.
- **[UBAH] Penerbitan NIM oleh admin**: konfigurasi `NIM_ISSUANCE_MODE=manual|auto` (default **manual** sesuai desain). Manual: `POST /student/admin/students/:id/issue-nim` (**Tetapkan NIM**) dan `POST /student/admin/nim/batch` (**Generate Batch Semua yang Lunas**, idempoten, mengembalikan ringkasan berhasil/dilewati beserta alasan). Mode `auto` = perilaku lama.
- **Format NIM** tampil di halaman dari konfigurasi NIM Service (`GET /nim/admin/formats/active`). Lihat pertanyaan Q2.
- **Gugus PKKMB** (baru, Student Service): tabel `pkkmb_groups` (`id`, `period_id`, `name`, `capacity`, `leader_name`, `schedule_note`) dan `pkkmb_assignments` (`student_id`, `group_id`, `assigned_by`, `method`). Endpoint: CRUD gugus; `POST /student/admin/pkkmb/auto-assign` (bagi rata per kapasitas, seimbangkan prodi dan jenis kelamin, hanya mahasiswa ber-NIM); ubah manual per mahasiswa; ekspor xlsx. Calon mahasiswa melihat gugusnya di `reregistration-overview` (field `pkkmb_group`).
- **Aktivasi akun SIAKAD**: di luar sistem ini. Sediakan ekspor `GET /student/admin/siakad-export?format=csv|xlsx` (NIM, nama, prodi, email, jalur, tahun masuk) dan kolom `siakad_exported_at`. Integrasi API SIAKAD menjadi CR tersendiri bila diminta.

### 2.6 Dashboard Institusional & Analitik Eksekutif (19)
`GET /admission/admin/institution-dashboard?period_id=&level=university|faculty|program&code=` (level mengikuti scope role: `pimpinan` semua, `dekan` fakultasnya, `prodi` prodinya):
- Kartu: **Kapasitas kuota total** (terisi / total kursi), **Tingkat kelulusan CBT** (% ≥ ambang), **Rasio keketatan** (pendaftar : kursi), **Total pendapatan PMB** (formulir + UKT, per kanal).
- Grafik **komparasi pendaftar & kuota per fakultas**; **rata-rata skor CBT & rapor per prodi**.
- **Kinerja antar fakultas**: per fakultas — daya tampung, pendaftar masuk, lulus seleksi, keterisian kuota %, daftar prodinya.

### 2.7 Pusat Pelaporan: Ekspor, Cetak Resmi, Broadcast, Buku Besar (20–23)
**Ekspor laporan (20):** tiga laporan, masing-masing `format=csv|xlsx` dan pratinjau tabel terpadu (`GET /reports/admin/preview?period_id=&program=`: nomor registrasi, nama, prodi, status formulir, skor CBT, status LoA, status UKT semester 1):
1. **Laporan Pendaftar PMB** — biodata, NISN, asal sekolah, pilihan prodi (Admission + Identity).
2. **Laporan Kelulusan Seleksi** — nominasi hasil CBT, nilai wawancara, nomor SK Rektor (Selection + Scoring).
3. **Laporan Rekonsiliasi Kas** — transaksi formulir & UKT per kanal (Finance).
Agregasi laporan lintas service ditempatkan di **Admission** (pemilik data pendaftar) dengan prefix `/admission/admin/reports/...`, kecuali laporan kas di Finance.

**Cetak dokumen resmi (21):** `GET .../reports/<jenis>/official.pdf` — kop institusi (nama yayasan, nama universitas, alamat, email, telepon, logo), judul, **Nomor Berita Acara** format `{urut 3 digit}/BA-PMB/{kode institusi}/{bulan romawi}/{tahun}` (urut per tahun, tidak berulang), tabel, kota & tanggal, jabatan penandatangan ("Ketua Panitia PMB {tahun}"), nama & NIP, cap waktu "Dicetak melalui Sistem PMB". Nomor dan metadata setiap cetakan disimpan (`official_documents`) supaya bisa dilacak.

**Broadcast notifikasi massal (22):** Notification Service, permission `broadcast.send`:
- `GET /notifications/admin/broadcast/templates` (template siap pakai, mis. "Pengingat Batas Waktu Pembayaran", "Instruksi Tahap Ujian CBT"), CRUD template untuk superadmin/admin_pmb.
- **Segmen target**: semua pendaftar, per status pendaftaran, per prodi, per jalur, belum bayar formulir, belum unggah berkas, lulus belum daftar ulang — `POST /notifications/admin/broadcast/preview` → jumlah penerima + contoh pesan yang sudah diisi variabel.
- **Variabel**: `{{NAMA_LENGKAP}}`, `{{NO_REGISTRASI}}`, `{{PRODI}}`, `{{JALUR}}`, `{{TAGIHAN}}`, `{{BATAS_BAYAR}}`, `{{JADWAL_CBT}}`; variabel tak dikenal → 422.
- **Kanal**: in-app, email, WhatsApp. WhatsApp memakai adapter `WhatsAppSender` (driver `log` default; driver penyedia resmi menyusul — lihat Q4).
- `POST /notifications/admin/broadcast` (kirim sekarang atau `scheduled_at`), dikirim bertahap lewat antrean (rate limit per kanal), status per penerima, riwayat broadcast `GET /notifications/admin/broadcast` + detail (terkirim/gagal).
- Pengaman: konfirmasi jumlah penerima di request (`expected_recipients` harus sama dengan hasil preview saat kirim), maks. 1 broadcast berjalan per admin, audit log.

**Buku besar kas masuk (23):** Finance, permission `finance.ledger.read`: `GET /finance/admin/ledger?period_id=&channel=&from=&to=` — kartu penerimaan per kanal/bank; tabel kronologis transaksi sukses (waktu transaksi, nomor invoice, nama, jenis, kanal, referensi, **kredit masuk**, saldo berjalan); ekspor csv/xlsx; sumber data: pembayaran sukses (Duitku terverifikasi + konfirmasi manual), dengan label sumber.

### 2.8 Master Data Organisasi: Fakultas & Program Studi (24)
- Kartu: fakultas aktif, total prodi, prodi yang membuka pendaftaran, format NIM aktif.
- Tabel prodi: kode prodi, **kode NIM**, nama, fakultas, jenjang, **akreditasi** (`Unggul`, `Baik Sekali`, `Baik`, `A`, `B`, `C`), **estimasi UKT/semester** (dari Finance), **status PMB** (Aktif/Tutup), aksi **Tutup/Buka Pendaftaran**.
- **[UBAH]** Organization: kolom `study_programs.admission_open` (bool) + endpoint `PATCH /organization/admin/study-programs/:code/admission` (permission `organization.program.manage`). Admission menolak pendaftaran/pindah jalur ke prodi tertutup (`422 PROGRAM_CLOSED`) dan menyembunyikannya di `GET /admission/options`.
- Ubah estimasi UKT dari halaman ini → Finance `PUT /finance/admin/fee-schemes/...` (permission `finance.fee.manage`); tampilan tabel dari endpoint agregat `GET /organization/admin/programs-overview` (Organization memanggil Finance internal).
- Filter fakultas, pencarian, tombol Tambah Program Studi (endpoint yang sudah ada).

### 2.9 Log aktivitas panitia (lintas service)
- Notification Service menambah `activity_events` (`id`, `period_id`, `actor_id`, `actor_name`, `actor_role`, `action`, `action_label`, `target_type`, `target_id`, `target_label`, `service`, `created_at`). Setiap service mengirim aktivitas admin penting lewat outbox ke `POST /internal/activity` (verifikasi berkas, konfirmasi manual, publikasi pengumuman, penerbitan NIM, broadcast, perubahan master, pindah jalur/pengunduran diri oleh admin).
- `GET /notifications/admin/activity?period_id=&limit=` (difilter scope/permission pembaca). Data pribadi sensitif (NIK, nominal detail) tidak dimasukkan ke label.

### 2.10 Profil institusi — field tambahan (Organization)
**[UBAH]** `institution_profile` menambah: `foundation_name` ("Yayasan …"), `document_code` (mis. `UBTH`), `city`, `phone`, `service_hours` (teks, mis. "Senin–Jumat 08.00–16.00 WIB | Sabtu 08.00–13.00 WIB"), `hotline`, `committee_chair_name`, `committee_chair_title`, `committee_chair_nip`, dan `bank_accounts` (daftar `{bank_name, account_number, account_name, is_primary}` untuk tab **Transfer Bank**). Semua diisi superadmin; seeder berisi contoh development.

---

## 3. CR-010 — Penyesuaian desain Portal Calon Mahasiswa v2 (halaman 05–12)

### 3.1 Beranda (05) **[UBAH dari CR-005 D.15.3]**
- Stepper menjadi **6 tahap** dengan urutan: `profile` (Profil — "Belum Lengkap"/"Lengkap"), `path` (Jalur Pendaftaran — label prodi), `payment` (Biaya Pendaftaran — nominal / "Lunas"), `documents` (Berkas Dokumen — "x/5 Diunggah"), `selection` (Proses Seleksi — mis. "Menunggu Bayar"), `reregistration` (Daftar Ulang). `completed_steps / total_steps` = x/6. Tahap boleh selesai tidak berurutan (mis. jalur selesai sebelum profil).
- `current_action` berisi "Langkah Aktif" (mis. "Lengkapi Formulir Biodata Pribadi") + tombol.
- **Jadwal & Agenda Seleksi** (baru): `agenda[]` dari data gelombang: `{no, title, description, date_label | deadline_label, state: "active"|"upcoming"|"done"}` — 1) Pengisian formulir biodata & berkas, 2) Pembayaran formulir (batas N hari kerja), 3) Ujian CBT & wawancara (tanggal), 4) Pengumuman kelulusan & LoA (tanggal), 5) Daftar ulang & pembayaran UKT semester 1 (masa registrasi). Butuh kolom gelombang: `exam_date`, `announcement_date`, `reregistration_start/end` (tambahkan bila belum ada).
- **Ringkasan dokumen & pembayaran** (baru): `summary_cards[]` — berkas (x dari 5 diunggah, aksi "Buka Berkas"), biaya formulir (status + nominal, aksi "Bayar Sekarang"), ujian CBT (status, aksi "Buka CBT").
- **Pusat layanan**: dari profil institusi (alamat, jam layanan, hotline, WhatsApp).
- `application.status_label` mengikuti tahap aktif (mis. "Pengisian Formulir Biodata").
- Kontrak `docs/contracts/dashboard.schema.json` diperbarui; field lama yang dihapus harus dicatat di `docs/API.md` bagian "Perubahan kontrak".

### 3.2 Biodata (06)
- Tambah isian **Rata-rata Nilai Rapor** di bagian Riwayat Asal Sekolah (lihat 2.4; wajib untuk jalur yang memakai komponen rapor, opsional untuk lainnya).
- Email & WhatsApp terisi otomatis dari akun (sudah di CR-005).

### 3.3 Jalur & Prodi (07)
- `GET /admission/options` menyertakan **biaya formulir per jalur** (`registration_fee`, `registration_fee_label`) dan deskripsi jalur; hanya prodi `admission_open = true`.
- Pilihan kedua: catatan aturan kuota (2.4).
- Status gabungan: `choices_status_label` mis. "Pilihan Tersimpan • Menunggu Pembayaran".
- Tombol **Batalkan Pilihan** = pembatalan CR-002; **Lanjut ke Pembayaran** → Biaya & Tagihan.

### 3.4 Berkas (08)
Mengikuti 5 persyaratan baru (2.2) dengan deskripsi per dokumen dari master (`description` wajib diisi di seeder: "Unggah pindaian ijazah atau SKL asli terlegalisasi", dst.), ringkasan "x dari 5 Berkas Diunggah", format PDF/JPG/PNG maks 2 MB. Pas foto di Profil dan di Berkas **saling mengisi** (K8 CR-005): unggahan di salah satu tempat tampil di keduanya.

### 3.5 Pembayaran (09)
- Tab **Virtual Account / Transfer Bank / QRIS**. `GET /finance/payment-methods` mengelompokkan `group` = `virtual_account|bank_transfer|qris` dan per metode: `bank_name`, `badge_label` ("Bank Utama BTH"/"Bank Mitra" — dari konfigurasi `PAYMENT_METHOD_BADGES`), `fee` (desain: "Biaya Transaksi Rp 0 (Gratis)" → konfigurasi `PAYMENT_FEE_BORNE_BY=merchant` default, sehingga biaya transaksi tidak dibebankan ke pendaftar).
- **Transfer Bank** = transfer manual ke rekening institusi (2.10 `bank_accounts`) + unggah bukti (endpoint yang sudah ada).
- **QRIS** = metode QRIS Duitku (respons berisi `qr_string`/`payment_url` sesuai dokumentasi Duitku; frontend menampilkan QR).
- Detail VA: nomor VA (salin), **nama akun** tampilan `PMB {short_name} - {nama lengkap}` (nama lengkap untuk tampilan; yang dikirim ke Duitku tetap dipotong ≤ 20 karakter), **batas pembayaran** — **[UBAH]** tagihan pendaftaran jatuh tempo **3 hari kerja** sejak pendaftaran dibuat (`REGISTRATION_PAYMENT_DUE_BUSINESS_DAYS=3`, Sabtu–Minggu tidak dihitung; tabel `holidays` opsional). Pembayaran Duitku kedaluwarsa mengikuti batas ini.
- Setelah jatuh tempo lewat: tagihan `expired`, pendaftaran tetap `awaiting_payment` dengan `payment_expired=true`; tombol "Buat tagihan baru" → `POST /invoices/me/:id/renew` (tagihan baru, batas baru; maks. 2 kali, lalu harus daftar ulang).
- Panel **Rincian Tagihan** + tombol **Cek Status Pembayaran** (endpoint polling CR-005) + tautan "Ingin mengganti program studi? Batalkan pendaftaran ini" (CR-002).
- **Informasi pembayaran** (teks dari backend, `payment_info_notes[]`): verifikasi otomatis, CBT terbuka setelah lunas, biaya tidak dapat ditarik kembali.

### 3.6 Proses Seleksi / CBT (10) **[UBAH]**
Desain: tombol CBT terkunci dengan teks **"Bayar Biaya Pendaftaran untuk Membuka Sesi"**, dan informasi pembayaran menyatakan **"Setelah pembayaran diverifikasi lunas, modul Ujian CBT Online akan langsung dibuka"**. Maka (keputusan K11):
- Kelayakan CBT = **biaya formulir lunas + biodata lengkap**; tidak menunggu verifikasi berkas. Exam menjadwalkan peserta otomatis ke sesi yang tersedia saat pembayaran lunas (bila ada sesi berkapasitas) — `can_start`/`start_block_reason_label` mengikuti syarat ini ("Bayar Biaya Pendaftaran untuk Membuka Sesi", "Lengkapi biodata terlebih dahulu", "Sesi belum dibuka").
- Verifikasi berkas berjalan **paralel**; **publikasi kelulusan** mensyaratkan berkas `verified` (pendaftar lulus tetapi berkas belum terverifikasi → status keputusan `plenary` dengan catatan "Berkas belum terverifikasi").
- State machine Admission disesuaikan (status `exam_scheduled` tidak lagi mensyaratkan `verified`); catat perubahan di dokumentasi state machine.

### 3.7 Pengumuman & Daftar Ulang (11)
Tidak berubah dari CR-005 D.15.9; tambahkan `pkkmb_group` (2.5) dan tanggal dari agenda gelombang.

### 3.8 Status pelunasan / kuitansi resmi (12)
- `GET /finance/me/invoices/:id/receipt-view` (JSON untuk halaman): judul "Pembayaran Formulir Terverifikasi Lunas", nomor transaksi/tagihan, nama, nomor registrasi, prodi, metode (mis. "Bank BSI Virtual Account"), waktu verifikasi, total, `lock_notice_label`, CTA berikutnya ("Lanjut ke Ujian CBT Online").
- `lock_notice_label`: desain menulis *"Pilihan jalur dan program studi telah terkunci dan tidak dapat diubah lagi"*. Karena CR-006 mengizinkan pindah jalur setelah bayar lewat persetujuan admin, teksnya diambil dari konfigurasi — default: "Pembayaran lunas. Pilihan jalur dan program studi terkunci; perubahan hanya melalui pengajuan ke panitia." (lihat Q1).

---

## 4. Keputusan default & pertanyaan untuk lead

**Keputusan default (catat di DECISIONS.md):**
| Kode | Keputusan |
|---|---|
| K11 | CBT terbuka setelah formulir lunas + biodata lengkap; verifikasi berkas paralel; kelulusan hanya bisa dipublikasikan bila berkas terverifikasi |
| K12 | NIM diterbitkan admin (satuan/batch), bukan otomatis (`NIM_ISSUANCE_MODE=manual`) |
| K13 | Konfirmasi lunas manual hanya keuangan/superadmin dengan bukti & nomor referensi mutasi unik |
| K14 | Ambang kelulusan global per gelombang (default 70.0); di bawah ambang → "Evaluasi Pleno", bukan otomatis tidak lulus |
| K15 | Batas bayar formulir 3 hari kerja, bisa diperpanjang maks. 2 kali |
| K16 | Role baru `dekan` (scope fakultas) untuk tab Dekanat di Dashboard Institusi |
| K17 | Biaya transaksi pembayaran ditanggung kampus (tampil Rp 0 ke pendaftar) |
| K18 | Integrasi SIAKAD lewat ekspor file dulu |

**Pertanyaan terbuka (default dipakai dulu):**
| No | Pertanyaan | Default |
|---|---|---|
| Q1 | Desain kuitansi: "pilihan terkunci dan tidak dapat diubah lagi", sedangkan CR-006 membolehkan pindah jalur setelah bayar lewat persetujuan. Mana yang dipakai? | CR-006 tetap berlaku; teks kuitansi dikonfigurasi |
| Q2 | Format NIM berbeda antar halaman: halaman 18 `[YY][Kode Prodi 2][Urut 4]`, halaman 24 `{YY}{NIM}{JALUR}{URUT}` | Template dapat dikonfigurasi; default tetap `{YY}{kode NIM prodi}{kode jalur}{urut 4}` (sesuai PRD & halaman 24) |
| Q3 | Nomor VA di desain berpola tetap (mis. `988 2026 01 94461`, turunan nomor registrasi) — apakah butuh **VA tetap per pendaftar** (produk Fixed VA Duitku / H2H bank) atau VA dinamis dari Duitku sudah cukup? | VA dinamis Duitku; nomor di desain dianggap contoh |
| Q4 | Penyedia WhatsApp Gateway resmi untuk broadcast? | Adapter `log`; driver penyedia ditambahkan setelah dipilih |
| Q5 | Pengaman konfirmasi lunas manual: perlu persetujuan dua orang keuangan? | Tidak (`MANUAL_CONFIRM_SECOND_APPROVAL=false`) |

---

## 5. Pengujian

### 5.1 Unit/integration
- RBAC: setiap endpoint admin dicek permission (tabel `docs/RBAC.md` diuji otomatis: untuk tiap endpoint, role tanpa permission → 403, role dengan permission → bukan 403); scope `prodi`/`dekan` memfilter data; klaim `perms` di token; header `X-User-Permissions` palsu dari klien dibuang gateway.
- Konfirmasi manual: nominal tidak cocok, referensi ganda, tagihan sudah lunas, tanpa bukti → ditolak; sukses → pembayaran Duitku pending dibatalkan.
- Batch NIM idempoten dan tanpa duplikat (uji konkurensi); auto-assign PKKMB menghormati kapasitas dan keseimbangan.
- Broadcast: variabel tak dikenal ditolak; `expected_recipients` tidak cocok ditolak; status per penerima.
- Nomor Berita Acara unik & berurutan per tahun (uji konkurensi).
- Jatuh tempo 3 hari kerja (lintas akhir pekan dan libur).
- Kelayakan CBT (K11) dan larangan publikasi bila berkas belum terverifikasi.

### 5.2 e2e (lewat gateway, DB kosong, `PAYMENT_PROVIDER=mock`)
1. Login `keuangan` → `/auth/me/permissions` hanya memuat modul keuangan; akses verifikasi berkas → 403; rekonsiliasi → 200.
2. Login `prodi` → overview & dashboard institusi hanya berisi prodinya; `dekan` → fakultasnya.
3. Alur calon mahasiswa v2: dashboard 6 tahap + agenda + ringkasan; bayar (mock) → CBT `can_start=true` walau berkas belum diverifikasi; submit & verifikasi berkas berjalan paralel.
4. Admin: antrean berkas dengan jumlah per status & pencarian NIK; terima/revisi per dokumen; `x/5 sah` benar.
5. Keuangan: konfirmasi lunas manual dengan bukti → lunas + kwitansi + log aktivitas; ulangi referensi yang sama → ditolak.
6. Seleksi: ambang 70 → nilai < 70 jadi "Evaluasi Pleno"; input nilai wawancara; publikasi dengan nomor SK; pendaftar lulus tapi berkas belum terverifikasi tidak bisa dipublikasikan.
7. NIM batch → hanya yang siap; auto-assign PKKMB; calon mahasiswa melihat gugusnya.
8. Laporan: tiga ekspor csv/xlsx; cetak resmi PDF memuat nomor berita acara berurutan.
9. Broadcast ke segmen "belum bayar formulir" → penerima tepat; notifikasi in-app & email masuk (Mailpit).
10. Buku besar memuat pembayaran mock & manual dengan saldo berjalan benar.
11. Master data: tutup pendaftaran prodi → hilang dari `options`, pendaftaran ke prodi itu → `PROGRAM_CLOSED`.
12. Semua skenario lama tetap lulus; ringkasan akhir tetap `Total PASS: X, FAIL: Y`.

Error code baru: `PROGRAM_CLOSED`, `INVOICE_ALREADY_PAID`, `DUPLICATE_BANK_REFERENCE`, `RECIPIENT_COUNT_MISMATCH`, `UNKNOWN_TEMPLATE_VARIABLE`, `PUBLISH_REQUIRES_VERIFIED_DOCUMENTS`, `PERMISSION_DENIED` (403 dengan kode permission yang dibutuhkan di `errors`).

---

## 6. Catatan untuk frontend (koordinasi dengan Wahhed)
- Arahan lead: **menu navigasi admin** jangan berbentuk kartu (terlihat seperti kartu informasi) — gunakan sidebar atau tab yang jelas; **pisahkan halaman admin dan halaman calon mahasiswa** di Figma (halaman/section terpisah).
- Menu admin dibangun dari `GET /auth/me/permissions` → `modules[]` (sembunyikan modul `allowed=false`, tampilkan mode baca bila `read_only=true`).
- Frame 22 (Broadcast) dan 23 (Buku Besar) pada ekspor tertutup modal cetak; bagian yang terlihat sudah dipakai di PRD ini. Bila ada elemen lain, laporkan sebagai CR baru.
- Belum ada desain untuk: riwayat & pindah jalur, pengunduran diri (CR-006/007), pengerjaan soal CBT, pengaturan RBAC oleh superadmin.

## 7. Definition of Done
- [ ] RBAC permission berjalan di semua endpoint admin; `docs/RBAC.md` lengkap dan teruji otomatis.
- [ ] Semua endpoint Bagian 2 dan 3 jadi, terdokumentasi (swagger, `docs/API.md` bagian "Portal Admin" dan "Perubahan kontrak", Postman, `docs/contracts/*.schema.json`), dan teruji.
- [ ] Skenario 5.2 masuk `scripts/e2e.sh` dan lulus bersama skenario lama.
- [ ] `docs/FE-BE-GAP.md` memuat halaman 05–24 dan tidak ada baris `Open`.
- [ ] CR-009 dan CR-010 `done` di `docs/CHANGE_REQUESTS.md`; K11–K18 dan Q1–Q5 di `docs/DECISIONS.md`.
- [ ] Class diagram & ERD semua service yang berubah diperbarui.
- [ ] `scripts/final-check.ps1` → AMAN.
