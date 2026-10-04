# Daftar Pelacakan Bug PMB (BUGS.md)

Dokumen ini mencatat seluruh bug yang ditemukan selama audit mandiri dan pengembangan sesuai PRD Bagian 21.1.

| ID | Tanggal | Sumber | Halaman/Endpoint | Langkah | Hasil | Harapan | Severity | Status | Commit |
|---|---|---|---|---|---|---|---|---|---|
| BUG-001 | 2026-10-04 | agent | `/finance/admin/overview` | Panggil ringkasan eksekutif finance | Field `num_terms` tidak terpetakan | Terpetakan dengan alias `num_terms` & `term_number` | P1 | Fixed | c89b6fa |
| BUG-002 | 2026-10-04 | agent | `/selection/admin/announcements/publish` | Publikasikan pengumuman kelulusan | Status verifikasi berkas tidak dikenali fleksibel | Mendukung status `verified` dan unmarshaling invoice | P1 | Fixed | a5e5eb3 |
| BUG-003 | 2026-10-04 | agent | `POST /auth/admin/users` | Pembuatan akun admin / panitia | Validasi gagal karena payload kurang `full_name` / `phone` | Payload menyertakan field wajib lengkap | P1 | Fixed | 35e93f6 |
| BUG-004 | 2026-10-04 | agent | `/exam/admin/participants/schedule` | Penjadwalan peserta ujian CBT | Gagal unmarshal respon internal pendaftaran | Unmarshaling respon internal fleksibel & robust | P1 | Fixed | 7be2b42 |
| BUG-005 | 2026-10-04 | agent | `POST /finance/invoices/:id/pay` | Inisiasi pembayaran via Duitku sandbox/produksi | URL POP dan WebAPI belum dinamis berdasarkan `DUITKU_ENV`, belum ada endpoint publik `/finance/payment-config` | `DUITKU_ENV=sandbox\|production` memilih URL otomatis, start guard `APP_ENV=production` | P1 | Fixed | 690a4f8 |
| BUG-006 | 2026-10-04 | agent | `POST /admission/applications/select-path` | Pilih jalur gratis (`registration_fee = 0`) | Masih menerbitkan invoice berbayar dan meminta pembayaran | Invoice Rp 0 otomatis lunas metode `free`, bypass step pembayaran, `409 INVOICE_FREE` | P1 | Fixed | 690a4f8 |
