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
