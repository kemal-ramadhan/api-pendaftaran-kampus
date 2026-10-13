# Laporan Pelaksanaan Eksekusi Agent PMB Backend

Dokumen ini mencatat kemajuan pelaksanaan menyeluruh instruksi kerja proyek PMB Backend sesuai arahan `PRD-TAMBAHAN-CR011.md`.

---

## 1. Fase R — Kesiapan Rilis PMB (CR-013)

- **Tanggal Pelaksanaan**: 4 Oktober 2026
- **Status Akhir**: **SELESAI (AMAN / LULUS 100%)**
- **Branch**: `main`
- **Tag Rilis**: `v0.5.1-rilis`
- **Hash Commit Rilis**: `57e6f3d`

### 1.1 Status Milestone Fase R

| Milestone | Ruang Lingkup | Status | Keterangan |
|---|---|---|---|
| **R0** | Audit Mandiri & Triase Bug | **Selesai** | Dibuat `docs/BUGS.md`, penelusuran log dan verifikasi seluruh skenario alur pendaftaran. |
| **R1** | Perbaikan Bug & Jalur Gratis | **Selesai** | Implementasi jalur pendaftaran Rp 0 (metode `free`), bypass otomatis pembayaran, pembatalan mandiri, dan pembuatan `scripts/smoke-camaba.sh` (< 3 menit, lulus 100%). |
| **R2** | Kesiapan Duitku Sandbox & Produksi | **Selesai** | Mode lingkungan `DUITKU_ENV`, startup guard produksi, rekonsiliasi berkala 10 menit, publik endpoint `/payment-config`, serta verifikasi `docs/DUITKU-SANDBOX.md` dan `scripts/duitku-sandbox-check.ps1`. |
| **R3** | Dokumen Operasional & Checklist Rilis | **Selesai** | Dibuat `docs/DUITKU-PRODUKSI.md`, `docs/DEPLOY.md`, `docs/RELEASE-CHECKLIST.md`, `docs/PANDUAN-ALUR.md`, `deploy/docker-compose.prod.yml`, diagram kelas & ERD api-gateway, verifikasi lulus `scripts/final-check.ps1 -Tahap rilis`. |

### 1.2 Ringkasan Pengujian & Hasil Audit

1. **Pemeriksaan Secret Repo**:
   - Bebas dari commit `.env`, private key, dan sertifikat sensitif.
   - Placeholder secret di dokumentasi telah disanitasi.
2. **Go Modules & Unit Tests**:
   - Seluruh 13 Go module (`pkg` + 12 service Go) `go mod tidy` bersih, `go vet` lulus, dan seluruh unit/integration test lulus 100%.
   - `ocr-service` pytest lulus.
3. **Smoke Test Camaba (`scripts/smoke-camaba.sh`)**:
   - 9 Skenario utama pendaftaran jalur berbayar dan jalur gratis lulus dalam waktu < 2 menit.
4. **E2E Test Komprehensif (`scripts/e2e.sh`)**:
   - Seluruh skenario alur (registrasi, verifikasi OTP, login, profil, KTP scan, biodata, berkas, pembayaran VA/transfer, seleksi, CBT, kelulusan, cicilan daftar ulang, UKT, NIM) lulus tanpa kegagalan (`FAIL: 0`).
5. **Verifikasi Final (`scripts/final-check.ps1 -Tahap rilis`)**:
   - Hasil: **AMAN - SEMUA PEMERIKSAAN LULUS**.

### 1.3 Perubahan Endpoint & Variabel Lingkungan

- **Endpoint Publik Baru**:
  - `GET /api/v1/payment-config` dan `GET /api/v1/finance/payment-config`: konfigurasi lingkungan pembayaran publik & URL pustaka POP JS.
  - `GET /api/v1/admission/paths`: mendukung atribut `registration_fee_label = "Gratis"` untuk jalur gratis.
- **Variabel Lingkungan yang Harus Disiapkan Tim di Produksi**:
  - `APP_ENV` (`production` / `development`)
  - `DUITKU_ENV` (`production` / `sandbox`)
  - `DUITKU_MERCHANT_CODE`
  - `DUITKU_API_KEY`
  - `DUITKU_CALLBACK_URL`
  - `DUITKU_RETURN_URL`
  - `ALLOW_SANDBOX_IN_PRODUCTION` (opsional untuk server demo)
  - `DEMO_PASSWORD` (untuk seeder data demo)

---

## 2. Fase K — Konsolidasi 3 Service (CR-011) & Dual-Auth/SSO (CR-012)

- **Tanggal Pelaksanaan**: 4–10 Oktober 2026
- **Status Akhir**: **SELESAI (LULUS 100%)**
- **Branch**: `main`
- **Tag Rilis**: `v0.6.0-3-services` & `v0.7.0-sso-ready`
- **Komponen**:
  - Konsolidasi 13 service menjadi 3 service (`gateway`, `pmb-service`, `ocr-service`) dan 2 database (`pmb_auth`, `pmb_app`).
  - Pemisahan model Auth: Versi A (SSO Core mandiri `cmd/sso`) dan Versi B (Auth PMB).
  - Skema dual-auth: `service_accounts`, `user_accounts`, `user_account_roles`, `signing_keys`.
  - Protokol OIDC Discovery (`/.well-known/openid-configuration`), OAuth2 Authorization Code dengan PKCE S256, JWKS, token revoke, logout, dan endpoint `/sso/login`.

---

## 3. Fase F — Modul Referal Guru BK & Komisi (CR-014)

- **Tanggal Pelaksanaan**: 11–13 Oktober 2026
- **Status Akhir**: **SELESAI (LULUS 100%)**
- **Branch**: `main`
- **Tag Rilis**: `v0.8.0-referal`
- **Komponen**:
  - Modul baru `services/pmb-service/internal/modules/referral` (arsitektur modular terisolasi).
  - Fitur pemasangan kode referal siswa dengan persetujuan (`consent_at`).
  - Verifikasi kolektif guru BK (`referral_collective`) yang melunasi tagihan formulir dan melepas pendaftar ke tahap berikutnya.
  - Snapshot komisi (persen/nominal), penanganan jalur gratis (Rp 0 tanpa komisi), dan mode penyelesaian komisi (`terpisah` / `potong_setoran`).
  - Setoran dana guru BK ke kampus dengan nomor referensi dan bukti transfer + verifikasi keuangan.
  - Impor akun PIC guru BK dari file CSV multi-baris.
  - Gateway proxy `/api/v1/referral` terkonfigurasi di `routes.yaml`.

---

## 4. Fase G — Penyesuaian Launching (CR-015)

- **Tanggal Pelaksanaan**: 13 Oktober 2026
- **Status Akhir**: **SELESAI (LULUS 100%)**
- **Branch**: `main`
- **Tag Rilis**: `v0.9.0-launching`
- **Komponen**:
  - **G1 (Direct Payment Duitku v2)**: API v2 inquiry tanpa popup, render QRIS base64 PNG dengan `go-qrcode`, instruksi per kanal.
  - **G2 (WhatsApp Group Onboarding)**: Endpoint onboarding dan gatekeeper `409 WA_GROUP_JOIN_REQUIRED`.
  - **G3 (NIM SK Rektor 065)**: Format otomatis `{FAKULTAS}{PRODI}{KKNI}{JALUR}{TAHUN2}{SEMESTER}{URUT:3}`, lulus 7/7 kasus uji resmi.
  - **G4 (Biodata SEVIMA & Ekspor XLSX)**: Generator 62 kolom format SEVIMA tipe string cell (mencegah eksponensial numerik NIK/KK/HP) dan validasi data.
  - **G5 (Identitas Dinamis & Audit Hardcode)**: Pembersihan hardcode institusi, profil dinamis di `GET /organization/institution-profile`.
  - **G6 (Dashboard Guru BK)**: Endpoint kartu performa mitra `GET /referral/partner/dashboard`.

---

## 5. Kesimpulan Status Akhir
Seluruh fase roadmap pada dokumen `PRD-TAMBAHAN-CR011.md` (Fase R, Fase K, Fase F, dan Fase G) telah **100% selesai dikerjakan, diuji, dan di-push ke GitHub**.
