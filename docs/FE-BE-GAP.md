# Frontend-Backend Gap Analysis (FE-BE-GAP.md)

Dokumen ini melacak kesenjangan antara desain antarmuka pengguna (Frontend Figma PMB karya Wahhed yang telah disetujui lead) dan ketersediaan kontrak API Backend PMB. Setiap kesenjangan dianalisis dan dipetakan ke endpoint, keputusan arsitektur, dan tahapan pengerjaan.

---

## Tabel Kesenjangan Frontend vs Backend (Berdasarkan PRD D.13)

| No | Halaman Figma | Kebutuhan Data & Interaksi | Endpoint Backend yang Ada | Kekurangan / Kesenjangan | Keputusan Arsitektur | Tahap |
|---|---|---|---|---|---|---|
| 1 | **Detail CBT & Pengerjaan Soal** | Alur ujian online interaktif: navigasi nomor soal, ragu-ragu, timer mundur tersinkronisasi server, autosave jawaban per soal, submit ujian, review esai oleh dosen | Hanya ada penjadwalan & kartu ujian di Exam Service (`/exam/me/schedule`, `/exam/me/card`) | Belum ada modul bank soal, paket ujian CBT, attempt peserta, autosave, dan timer server | Implementasi penuh CR-003 CBT di Exam Service & integrasi nilai ke Scoring Service | Tahap 6 & 7 |
| 2 | **Batalkan / Daftar Lagi** | Tombol batalkan pendaftaran sebelum bayar, dialog konfirmasi, riwayat pendaftaran lengkap, tombol daftar lagi setelah `failed`, penonaktifan opsi jalur/prodi yang sudah gagal | `GET /admission/applications/me`, `POST /admission/applications` | Belum ada endpoint pembatalan, riwayat multi-pendaftaran, dan pemeriksaan kelayakan daftar ulang (`eligibility`) | Implementasi CR-002: status `cancelled`, endpoint `/applications/me/cancel`, `/history`, `/eligibility`, guard `SAME_CHOICE_ALREADY_FAILED` | Tahap 3, 4, 5, 8 |
| 3 | **Halaman Detail Pembayaran** | Rincian detail tagihan, pilihan metode pembayaran, nomor VA / link pembayaran Duitku POP, batas waktu (countdown), riwayat status real-time, link unduh kwitansi PDF | `GET /invoices/me`, `POST /payments/va` | Detail invoice belum memuat informasi pembayaran aktif secara agregat; belum ada endpoint polling status instan saat redirect kembali dari Duitku | Perluas respons `GET /invoices/me/:id` dengan detail komprehensif (active payment info, receipt link, installment). Tambahkan `GET /invoices/me/:id/payments/:payment_id/status` untuk polling status | Tahap 4 |
| 4 | **Dashboard Calon Mahasiswa** | Stepper alur pendaftaran 5 tahap (Pilih Jalur & Prodi -> Pembayaran -> Berkas & Biodata -> Ujian/Seleksi -> Pengumuman & Daftar Ulang) dengan status `done`, `current`, `locked` | Status pendaftaran tunggal di `GET /applications/me` | Frontend harus menghitung sendiri logika stepper yang rumit dan rentan inkonsisten | Buat endpoint agregasi stepper di Admission Service: `GET /admission/applications/me/progress` yang mengembalikan langkah terstruktur beserta link aksi selanjutnya | Tahap 3 |
| 5 | **Semua Halaman Daftar / Tabel** | Paginasi fleksibel, pencarian kata kunci, filter status/gelombang/prodi, metadata total item & halaman | Sebagian endpoint memakai parameter `limit` & `offset` | Struktur pembungkus respons dan metadata halaman perlu diseragamkan | Standarisasi seluruh endpoint list menggunakan objek `meta` (`total`, `page`, `limit`) sesuai PRD C.5.6 | Semua Tahap |
| 6 | **Tampilan Mobile Responsif** | Tampilan bersih pada layar ponsel calon mahasiswa | Format JSON standar | Tidak ada perubahan arsitektur backend | Backend mengoptimalkan payload respons agar ringkas dan efisien tanpa mengirim data berlebih | Frontend (Wahhed) |

---

## Detail Spesifikasi Endpoint Tambahan

### 1. Stepper Progres Calon Mahasiswa (`GET /admission/applications/me/progress`)
Mengembalikan status langkah flowchart:
```json
{
  "code": "SUCCESS",
  "message": "Progres pendaftaran berhasil diambil",
  "data": {
    "application_id": "uuid",
    "status": "awaiting_payment",
    "current_step": 2,
    "steps": [
      { "step": 1, "title": "Pilih Gelombang & Program Studi", "status": "done" },
      { "step": 2, "title": "Pembayaran Biaya Pendaftaran", "status": "current", "action_url": "/pembayaran" },
      { "step": 3, "title": "Pengisian Biodata & Berkas", "status": "locked" },
      { "step": 4, "title": "Pelaksanaan Ujian / Seleksi", "status": "locked" },
      { "step": 5, "title": "Pengumuman & Daftar Ulang", "status": "locked" }
    ]
  }
}
```

### 2. Polling Status Pembayaran (`GET /invoices/me/:id/payments/:payment_id/status`)
Dipakai oleh frontend saat pengguna kembali dari halaman payment gateway Duitku POP:
```json
{
  "code": "SUCCESS",
  "message": "Status pembayaran berhasil diperiksa",
  "data": {
    "invoice_id": "uuid",
    "payment_id": "uuid",
    "status": "success",
    "paid_at": "2026-09-25T15:30:00Z",
    "amount": 250000,
    "is_settled": true
  }
}
```

---

## Tabel Kesenjangan Desain Figma PMB v2 Halaman 05–24 (CR-009 & CR-010)

| No | Halaman Figma | Kategori | Kebutuhan Data & Desain UI | Status | Endpoint / Rencana Backend | Tahap |
|---|---|---|---|---|---|---|
| 05 | **Halaman 05 — Beranda Camaba v2** | Portal Camaba | Stepper 6 tahap (`profile`, `path`, `payment`, `documents`, `selection`, `reregistration`), agenda & jadwal seleksi, summary cards, status label, kontak institusi | Open | `GET /admission/applications/me/dashboard` (v2), `GET /admission/options` | Tahap 14 / CR-010 |
| 06 | **Halaman 06 — Biodata Pribadi & Sekolah** | Portal Camaba | Isian rata-rata rapor `report_average` di riwayat asal sekolah | Open | `PUT /identity/me/education`, `GET /identity/me/profile` | Tahap 14 / CR-010 |
| 07 | **Halaman 07 — Jalur & Pilihan Program Studi** | Portal Camaba | Pilihan prodi hanya `admission_open = true`, biaya formulir per jalur di options, catatan kuota pilihan kedua, status gabungan | Open | `GET /admission/options`, `POST /admission/applications` | Tahap 14 / CR-010 |
| 08 | **Halaman 08 — Berkas Dokumen Persyaratan** | Portal Camaba | 5 dokumen wajib (IJAZAH_SKL, KTP_KIA_KP, KK, PAS_FOTO, SURAT_SEHAT), sinkronisasi pas foto profil & berkas | Open | `GET /admission/applications/me/documents`, `POST /admission/applications/me/documents` | Tahap 14 / CR-010 |
| 09 | **Halaman 09 — Biaya & Tagihan Formulir** | Portal Camaba | Tab Virtual Account / Transfer Bank / QRIS, badge bank mitra/utama, fee transaksi gratis ditanggung kampus, jatuh tempo 3 hari kerja, renew tagihan | Open | `GET /finance/payment-methods`, `POST /invoices/me/:id/renew` | Tahap 14 / CR-010 |
| 10 | **Halaman 10 — Pelaksanaan Ujian CBT** | Portal Camaba | Kelayakan CBT K11 (bayar lunas + biodata lengkap buka modul CBT tanpa tunggu verifikasi berkas), status terkunci/terbuka | Open | `GET /exam/cbt/me/eligibility`, `GET /exam/cbt/me/attempts` | Tahap 14 / CR-010 |
| 11 | **Halaman 11 — Pengumuman & Registrasi Ulang** | Portal Camaba | Hasil kelulusan, jadwal registrasi, penugasan gugus PKKMB | Open | `GET /selection/me/announcement`, `GET /student/me/reregistration-overview` | Tahap 14 / CR-010 |
| 12 | **Halaman 12 — Kuitansi Resmi Formulir** | Portal Camaba | Status pelunasan lunas, rincian kuitansi resmi, teks penguncian pilihan jalur | Open | `GET /finance/me/invoices/:id/receipt-view` | Tahap 14 / CR-010 |
| 13 | **Halaman 13 — Ringkasan Eksekutif & KPI** | Portal Admin | Kartu overview pendaftar, verifikasi berkas, penerimaan kas, lulus seleksi, funnel 6 tahap, donat peminat prodi, log aktivitas | Open | `GET /admission/admin/overview?period_id=` | Tahap 14 / CR-009 |
| 14 | **Halaman 14 — Antrean Verifikasi Berkas** | Portal Admin | Antrean berkas dengan filter status & counter, pencarian NIK ter-hash, indikator x/5 sah | Open | `GET /admission/admin/document-queue` | Tahap 14 / CR-009 |
| 15 | **Halaman 15 — Modal Review Dokumen** | Portal Admin | Tab 5 dokumen, preview presigned url, terima/revisi berkas, navigasi berikutnya | Open | `GET /admission/admin/applications/:id/documents` | Tahap 14 / CR-009 |
| 16 | **Halaman 16 — Rekonsiliasi Keuangan & Virtual Account** | Portal Admin | Metrik formulir & UKT, proporsi dana & kanal, daftar tagihan, ekspor csv/xlsx, konfirmasi lunas manual referensi mutasi unik | Open | `GET /finance/admin/reconciliation`, `POST /finance/admin/invoices/:id/manual-confirm` | Tahap 14 / CR-009 |
| 17 | **Halaman 17 — Seleksi Akademik & CBT** | Portal Admin | Scoreboard, ambang batas global 70.0 (K14), status plenary/evaluasi pleno, input nilai wawancara, SK kelulusan rektor, filter pilihan kedua | Open | `GET /selection/admin/scoreboard`, `PUT /scoring/admin/interview-scores/:application_id` | Tahap 14 / CR-009 |
| 18 | **Halaman 18 — Penetapan NIM & Registrasi Maba** | Portal Admin | Enrolment list, NIM manual/batch K12, format NIM aktif, pembagian gugus PKKMB, ekspor SIAKAD | Open | `GET /student/admin/enrolment`, `POST /student/admin/nim/batch`, `/pkkmb` | Tahap 14 / CR-009 |
| 19 | **Halaman 19 — Dashboard Institusional & Analitik Eksekutif** | Portal Admin | Kapasitas kuota, tingkat kelulusan CBT, rasio keketatan, komparasi per fakultas & prodi, hierarki universitas/fakultas/prodi | Open | `GET /admission/admin/institution-dashboard` | Tahap 14 / CR-009 |
| 20 | **Halaman 20 — Pusat Pelaporan: Ekspor Laporan** | Portal Admin | Laporan Pendaftar PMB, Kelulusan Seleksi, Rekonsiliasi Kas, format csv/xlsx, pratinjau tabel | Open | `GET /admission/admin/reports/...`, `GET /reports/admin/preview` | Tahap 14 / CR-009 |
| 21 | **Halaman 21 — Cetak Dokumen Resmi Berita Acara** | Portal Admin | Kop institusi resmi, nomor Berita Acara unik per tahun `{urut 3 digit}/BA-PMB/...`, tanda tangan ketua panitia, unduh PDF | Open | `GET /admission/admin/reports/:type/official.pdf` | Tahap 14 / CR-009 |
| 22 | **Halaman 22 — Broadcast Notifikasi Massal** | Portal Admin | Template pesan, segmentasi target pendaftar, variabel pesan dinamis, estimasi penerima, kanal in-app/email/WhatsApp, riwayat pengiriman | Open | `GET/POST /notifications/admin/broadcast/...` | Tahap 14 / CR-009 |
| 23 | **Halaman 23 — Buku Besar Kas Masuk** | Portal Admin | Penerimaan kas kronologis, kredit masuk, saldo berjalan, filter kanal dan periode, sumber Duitku vs manual | Open | `GET /finance/admin/ledger` | Tahap 14 / CR-009 |
| 24 | **Halaman 24 — Master Data Organisasi: Fakultas & Program Studi** | Portal Admin | Daftar prodi, akreditasi, estimasi UKT, format NIM, buka/tutup pendaftaran `admission_open` | Open | `GET /organization/admin/programs-overview`, `PATCH /organization/admin/study-programs/:code/admission` | Tahap 14 / CR-009 |

