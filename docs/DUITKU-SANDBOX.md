# Panduan Pengujian & Integrasi Duitku Sandbox

| | |
|---|---|
| **Dokumen** | Panduan Resmi Integrasi Duitku Sandbox (CR-008) |
| **Target Layanan** | `services/finance-service`, `services/api-gateway`, `scripts/duitku-sandbox-check.ps1` |
| **Tanggal Verifikasi** | 29 September 2026 |
| **Dokumentasi Resmi** | [docs.duitku.com](https://docs.duitku.com/) |

---

## 1. Verifikasi Endpoint & Rumus Signature Resmi

Berdasarkan pengecekan dokumentasi resmi Duitku per **29 September 2026**, integrasi Duitku menggunakan dua host terpisah untuk layanan POP (Payment Optimization Platform) dan WebAPI.

| Fungsi | Host & Endpoint Sandbox | Rumus Signature |
|---|---|---|
| **Create Invoice (POP)** | `https://api-sandbox.duitku.com/api/merchant/createInvoice` | Header `x-duitku-signature = HMAC_SHA256(merchantCode + timestampMs, apiKey)` |
| **Daftar Metode Pembayaran (WebAPI)** | `https://sandbox.duitku.com/webapi/api/merchant/paymentmethod/getpaymentmethod` | Body `signature = SHA256(merchantcode + amount + datetime + apiKey)` *(format datetime: `YYYY-MM-DD HH:mm:ss`)* |
| **Cek Status Transaksi (WebAPI)** | `https://sandbox.duitku.com/webapi/api/merchant/transactionStatus` | Body `signature = MD5(merchantCode + merchantOrderId + apiKey)` |
| **Callback Notifikasi Pembayaran** | Ke `DUITKU_CALLBACK_URL` | Mendukung `MD5(merchantCode + amount + merchantOrderId + apiKey)` dan `HMAC_SHA256(merchantCode + amount + merchantOrderId, apiKey)` *(diverifikasi secara constant-time)* |

### Pemisahan Base URL Konfigurasi
Untuk fleksibilitas konfigurasi antara lingkungan sandbox dan production, disediakan variabel lingkungan terpisah:
- `DUITKU_POP_BASE_URL` (default sandbox: `https://api-sandbox.duitku.com`)
- `DUITKU_WEBAPI_BASE_URL` (default sandbox: `https://sandbox.duitku.com`)
- `DUITKU_ENV` (`sandbox` atau `production`)

---

## 2. Pemetaan Error & Keamanan

1. **Pemetaan Error ke 502**:
   - Jika Duitku mengembalikan status HTTP non-200 atau `statusCode` selain `00`, sistem mengembalikan respon HTTP `502 Bad Gateway` dengan kode `PAYMENT_PROVIDER_ERROR` dan pesan dalam Bahasa Indonesia:
     `"Layanan penyedia pembayaran sedang mengalami gangguan"`.
   - Kode error dan pesan asli dari Duitku hanya dicatat ke logger server internal (Zap Logger) dan **tidak pernah membocorkan API Key atau kredensial sensitif**.
2. **Keamanan Callback**:
   - Verifikasi signature dilakukan secara `subtle.ConstantTimeCompare` untuk mencegah timing attack.
   - Pengecekan kecocokan `merchantCode` dan integritas nilai `amount`.
   - Disediakan opsi allowlist IP pengirim webhook: `DUITKU_CALLBACK_IP_ALLOWLIST` (default kosong = nonaktif).
   - Seluruh payload mentah dicatat ke tabel `payment_webhook_logs` untuk keperluan audit dan rekonsiliasi.
3. **Penyelarasan Interval Rekonsiliasi**:
   - Interval background job rekonsiliasi dapat dikonfigurasi melalui `DUITKU_RECONCILE_INTERVAL` (default production: `10m`, untuk keperluan pengujian dapat disetel ke `1m`).

---

## 3. Panduan Langkah Demi Langkah Pengujian Sandbox (Untuk Tim / Diar)

### Langkah A: Persiapan Kredensial Sandbox
1. Minta kredensial sandbox tim PMB ke Pak Kemal, atau daftar akun baru di dashboard sandbox Duitku ([sandbox.duitku.com](https://sandbox.duitku.com)).
2. Dapatkan **Merchant Code** dan **API Key** dari menu *Project / Pengaturan Proyek*.
3. Simpan kredensial **hanya di file `.env` lokal**. Jangan pernah melakukan commit file `.env` atau mencetak API key di repositori git!

Contoh baris konfigurasi di `.env`:
```env
PAYMENT_PROVIDER=duitku
DUITKU_ENV=sandbox
DUITKU_MERCHANT_CODE=DXXXX
DUITKU_API_KEY=<isi_api_key_sandbox_anda>
DUITKU_EXPIRY_MINUTES=1440
DUITKU_RECONCILE_INTERVAL=1m
```

### Langkah B: Menjalankan Tunnel Callback Publik
Duitku memerlukan URL HTTPS publik yang dapat diakses dari internet untuk mengirimkan HTTP callback:
Gunakan Cloudflare Tunnel tanpa perlu membuat akun:
```powershell
cloudflared tunnel --url http://localhost:8000
```
Salin URL publik yang dihasilkan (misalnya `https://abc-123.trycloudflare.com`), lalu set variabel:
```env
DUITKU_CALLBACK_URL=https://abc-123.trycloudflare.com/api/v1/finance/webhooks/payments/duitku
```

### Langkah C: Menjalankan Script Uji Otomatis Windows
Jalankan script PowerShell verifikasi sandbox:
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\duitku-sandbox-check.ps1
```

Alur yang dijalankan oleh script:
1. Memeriksa kesehatan API Gateway dan konektivitas endpoint `DUITKU_CALLBACK_URL`.
2. Mendaftarkan user calon mahasiswa baru dan melakukan aktivasi via OTP Mailpit.
3. Membuat aplikasi pendaftaran baru (`form_open`/`awaiting_payment`).
4. Mengambil daftar metode pembayaran real-time dari Duitku Sandbox WebAPI (`GET /api/v1/finance/payment-methods`).
5. Menginisialisasi pembayaran Virtual Account (`POST /api/v1/invoices/me/:id/pay`).
6. Menampilkan Nomor VA, Nama Bank, dan link pembayaran simulator di layar.
7. Menunggu pembayaran di simulator Duitku, melakukan polling status setiap 10 detik.
8. Melaporkan status pelunasan (apakah lunas via callback webhook real-time atau via rekonsiliasi).
9. Menyimpan hasil laporan ke file `duitku-sandbox-report.txt` (nomor merchant dan key disamarkan/masking).

### Langkah D: Uji Mode Rekonsiliasi (Opsional)
Untuk menguji bahwa pelunasan tetap berjalan saat webhook internet terputus (rekonsiliasi otomatis):
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\duitku-sandbox-check.ps1 -TestReconcile
```

### Langkah E: Kembali ke Mode Mock
Setelah pengujian sandbox selesai, kembalikan konfigurasi lokal ke mode mock agar seluruh skenario automated test CI/CD tetap berjalan tanpa ketergantungan internet:
```env
PAYMENT_PROVIDER=mock
```
Jalankan kembali `./scripts/e2e.sh` untuk memastikan suite end-to-end lengkap tetap PASS 100%.
