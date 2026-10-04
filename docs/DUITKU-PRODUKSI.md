# Panduan Integrasi dan Operasional Duitku Siap Pakai (Produksi)

Dokumen ini merupakan panduan operasional integrasi gateway pembayaran Duitku untuk PMB Backend pada lingkungan produksi sesuai mandat **CR-013**.

---

## 1. Persiapan Akun & Legalitas Merchant

1. **Registrasi Merchant Resmi**:
   - Daftarkan institusi kampus melalui portal [Duitku Merchant Portal](https://passport.duitku.com/).
   - Pilih jenis badan usaha: **Institusi Pendidikan / Yayasan / Perguruan Tinggi**.
2. **Unggah Dokumen Legalitas**:
   - SK Pendirian Perguruan Tinggi / Izin Operasional Kemendikbudristek.
   - NPWP Yayasan / Institusi.
   - KTP Penanggung Jawab / Pimpinan Institusi.
   - Rekening koran bank operasional penampungan pendaftaran.
3. **Pengaktifan Kanal Pembayaran**:
   - Virtual Account Bank Utama (BCA, Mandiri, BNI, BRI, Permata, CIMB).
   - QRIS Statis & Dinamis.
   - Gerai Ritel (Alfamart / Indomaret jika diaktifkan).

---

## 2. Konfigurasi Lingkungan Produksi

Pastikan variabel lingkungan pada `.env` server produksi terpasang dengan nilai riil dari dashboard Duitku:

```env
PAYMENT_PROVIDER=duitku
DUITKU_ENV=production
DUITKU_MERCHANT_CODE=DXXXX                  # Kode merchant resmi dari Duitku
DUITKU_API_KEY=<kunci_api_resmi_duitku>     # Production API Key
DUITKU_EXPIRY_MINUTES=1440                 # Batas waktu tagihan VA (default 24 jam)
DUITKU_CALLBACK_URL=https://pmb.kampus.ac.id/api/v1/finance/webhooks/payments/duitku
DUITKU_RETURN_URL=https://pmb.kampus.ac.id/dashboard/pembayaran/status
DUITKU_CALLBACK_IP_ALLOWLIST=             # Daftar IP resmi Duitku jika menggunakan filtering IP
ALLOW_SANDBOX_IN_PRODUCTION=false         # Wajib false di lingkungan produksi
RECONCILIATION_INTERVAL_MINUTES=10        # Rekonsiliasi otomatis setiap 10 menit
```

> [!CAUTION]
> Backend memicu **Startup Guard**: Service akan menolak menyala jika `APP_ENV=production` dan `PAYMENT_PROVIDER=duitku`, namun `DUITKU_ENV` bukan `production` atau kredensial kosong.

---

## 3. Whitelist IP & Jaringan

1. Daftarkan IP Publik server kampus pada menu **Pengaturan IP Server** di portal Duitku.
2. Pastikan firewall inbound membuka port HTTPS 443 untuk domain `https://pmb.kampus.ac.id` agar webhook Duitku dapat diterima tanpa kendala.

---

## 4. Konfigurasi Callback & Return URL di Dashboard Duitku

1. Masuk ke dashboard Duitku > **Pengaturan Proyek / Integrasi**.
2. Set **Callback URL**:
   `https://pmb.kampus.ac.id/api/v1/finance/webhooks/payments/duitku`
3. Set **Return URL**:
   `https://pmb.kampus.ac.id/dashboard/pembayaran/status`
4. Format Callback: `application/x-www-form-urlencoded` atau `application/json` (PMB Finance Service mendukung keduanya secara otomatis).

---

## 5. Prosedur Rotasi Kunci Rahasia (API Key Rotation)

1. Generate API Key baru pada dashboard Duitku di tab **API Keys**.
2. Perbarui variabel `DUITKU_API_KEY` pada file `.env` produksi.
3. Lakukan restart container finance:
   ```bash
   docker compose -f deploy/docker-compose.prod.yml restart finance
   ```
4. Verifikasi koneksi dengan menjalankan:
   ```powershell
   powershell -ExecutionPolicy Bypass -File scripts/duitku-sandbox-check.ps1 -Env production -GatewayUrl https://pmb.kampus.ac.id
   ```
5. Revoke API Key lama di dashboard Duitku setelah seluruh kanal terkonfirmasi aktif.

---

## 6. Prosedur Eskalasi Masalah Callback & Pembayaran Gantung

Jika calon mahasiswa telah membayar tetapi status tagihan masih `unpaid`:
1. **Pengecekan Otomatis (Background Reconcile)**:
   - Worker rekonsiliasi berjalan otomatis setiap 10 menit dan akan mengecek status transaksi ke WebAPI Duitku (`/merchant/transactionStatus`).
2. **Rekonsiliasi Manual Staf Keuangan**:
   - Staf keuangan dapat menekan tombol **"Sinkronisasi Status Pembayaran"** pada panel Admin Keuangan (`POST /api/v1/finance/admin/reconcile-pending`).
3. **Pemeriksaan Webhook Log**:
   - Cek database tabel `payment_webhook_logs` untuk melihat histori payload dan signature dari Duitku.
4. **Kontak Dukungan Duitku**:
   - Jika transaksi tidak tercatat di Duitku, hubungi tim support teknis Duitku dengan menyertakan Merchant Code dan Publisher Order ID / Reference.
