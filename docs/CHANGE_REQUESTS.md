# Change Requests Log (CHANGE_REQUESTS.md)

Dokumen ini mencatat seluruh permintaan perubahan (*Change Request* / CR) yang diajukan oleh lead (Pak Kemal) atau pemangku kepentingan selama pengembangan Sistem PMB Fase 2. Setiap perubahan dianalisis dampaknya terhadap arsitektur, basis data, endpoint, pengujian, dan tahapan timeline sebelum diimplementasikan.

---

## Tabel Log Change Request

| No. CR | Tanggal Pengajuan | Ringkasan Perubahan | Sumber / Pengaju | Service / Komponen Terdampak | Tahap Terdampak | Status | Dokumen Terkait |
|---|---|---|---|---|---|---|---|
| **CR-001** | 2026-09-24 | Integrasi Payment Gateway Duitku (Duitku POP / `createInvoice`) menggantikan mock-only; MockProvider tetap dipertahankan untuk pengujian otomatis | Pak Kemal Ramadhan (Lead / Product Owner) | Finance Service (`services/finance-service`), API Gateway (`services/api-gateway`), Compose (`deploy/docker-compose.yml`), Scripts (`scripts/e2e.sh`) | Tahap 4 (Finance 1) & Tahap 9 (Finance 2) | **Implemented (Tahap 4)** | `PRD-LENGKAP.md` Bagian D, `docs/DECISIONS.md` |

---

## Catatan Analisis Dampak & Riwayat CR

### CR-001 — Payment Gateway Duitku (24 September 2026)

#### 1. Latar Belakang & Pernyataan Kebutuhan
Lead memutuskan: *"Yang sudah ada kita pakai duitku.com, coba pelajari buat payment gatewaynya."*
Perubahan ini menggantikan spesifikasi awal PMB Fase 2 yang hanya merencanakan adapter *mock-only*. Integrasi nyata dengan Duitku POP (`createInvoice` & callback webhook) wajib tersedia dalam mode sandbox, sementara `MockProvider` tetap dipertahankan untuk kebutuhan automated test dan CI/CD.

#### 2. Dampak Komponen & Arsitektur
- **Finance Service (`services/finance-service`)**:
  - Penambahan `DuitkuProvider` mengimplementasikan interface `PaymentProvider` (`CreatePayment`, `ParseWebhook`, `VerifySignature`, `CheckStatus`).
  - Pemilihan provider secara dinamis via environment variable `PAYMENT_PROVIDER=mock|duitku`.
  - Penyesuaian `merchantOrderId` menggunakan ID pembayaran internal (unik).
  - Skema database `pmb_finance`: pencatatan `payment_webhook_logs`, penyimpanan `provider_ref`, `payment_url`, `va_number`, `payment_code`.
  - Pekerjaan latar belakang (reconciliation job) setiap 10 menit untuk memverifikasi pembayaran `pending` yang melebihi 15 menit menggunakan `CheckStatus`.
- **API Gateway (`services/api-gateway`)**:
  - Penambahan rute publik callback webhook: `POST /api/v1/finance/webhooks/payments/duitku`.
  - Rute ini bersifat publik (tanpa JWT dari klien), namun Gateway wajib membubuhkan HMAC signature internal saat meneruskan ke `finance-service`.
  - Konfigurasi proteksi rate limit terpisah dan pembatasan ukuran payload maksimal 64 KB.
- **Konfigurasi Environment**:
  - Variabel baru: `DUITKU_ENV`, `DUITKU_MERCHANT_CODE`, `DUITKU_API_KEY`, `DUITKU_EXPIRY_MINUTES`, `DUITKU_CALLBACK_URL`, `DUITKU_RETURN_URL`, `DUITKU_HTTP_TIMEOUT`.
- **Pengujian**:
  - Unit test DuitkuProvider menggunakan `httptest.Server` lokal tanpa memanggil endpoint riil Duitku.
  - Script pengujian manual sandbox: `scripts/duitku-sandbox-check.sh`.
