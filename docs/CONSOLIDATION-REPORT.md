# Laporan Konsolidasi Layanan PMB Backend (CR-011)

## 1. Ringkasan Eksekutif
Berdasarkan Change Request CR-011, PMB Backend telah dikonsolidasi secara menyeluruh dari arsitektur 13 microservices terdistribusi menjadi **3 layanan mandiri** dengan **2 basis data PostgreSQL**. Konsolidasi ini memangkas overhead latensi jaringan, meniadakan kegagalan JWKS discovery round-trip, menyederhanakan orkestrasi deployment, dan tetap menjamin isolasi batas modul bisnis secara ketat.

---

## 2. Metrik Perubahan

| Parameter | Sebelum Konsolidasi | Sesudah Konsolidasi (CR-011) | Dampak |
|---|---|---|---|
| **Jumlah Layanan Aplikasi** | 13 Services | **3 Services** (`gateway`, `pmb-service`, `ocr-service`) | Berkurang 77% |
| **Jumlah Basis Data PostgreSQL** | 11 Databases | **2 Databases** (`pmb_auth`, `pmb_app`) | Efisiensi koneksi pool & resource |
| **Port Publik** | :8000, :8080 - :8092 | **:8000** (Proxy & Auth), **:8081** (Internal Auth) | 1 Entry point publik |
| **Validasi JWT di Gateway** | HTTP JWKS round-trip ke auth-service | In-process Public Key RSA cache lokal | Latensi < 1ms |
| **Penyimpanan Domain PMB** | 10 Database terpisah | 1 Database `pmb_app` (10 Skema Postgres) | Schema isolation tanpa join cross-schema |
| **Background Runner** | Cron tersebar di 10 service | 1 Runner terpusat ber-advisory lock (`pg_try_advisory_lock`) | Mencegah race condition multi-instance |

---

## 3. Detail Implementasi Layanan

### A. `services/gateway`
- **Gabungan dari**: `api-gateway` dan `auth-service`.
- **Fungsi Utama**:
  1. Melayani autentikasi pengguna in-process (Login, Register, Refresh Token, Reset Password, OTP, Audit Log).
  2. Memverifikasi JWT secara langsung menggunakan public key RSA in-process.
  3. Reverse proxy ke upstream `pmb-service` (`:8080`) dan `ocr-service` (`:8089`).
  4. Menandatangani request downstream dengan HMAC SHA-256 (`X-Gateway-Signature`) serta menyematkan header identitas pengguna (`X-User-ID`, `X-User-Roles`, `X-User-Permissions`).
- **Database**: `pmb_auth`.

### B. `services/pmb-service`
- **Gabungan dari 10 service**: `organization`, `identity`, `notification`, `admission`, `finance`, `exam`, `scoring`, `selection`, `student`, `nim`.
- **Arsitektur Modular Monolith**:
  - Tiap modul memiliki package publik (`api.go`, `module.go`) dan package internal (`internal/`).
  - Aturan isolasi arsitektur divalidasi secara otomatis melalui `arch_test.go` untuk memastikan tidak ada cross-import antar modul internal.
  - Skema database diisolasi per modul (`organization.*`, `identity.*`, dsb.) menggunakan GORM dynamic naming strategy prefix.
- **Database**: `pmb_app`.

### C. `services/ocr-service`
- Layanan Python FastAPI berdedikasi untuk tugas pemrosesan gambar dan ekstraksi teks KTP menggunakan Tesseract / TrOCR.

---

## 4. Status Pengujian & Verifikasi
- Kompilasi `services/gateway`: **LULUS** (`go build ./...` exit 0).
- Kompilasi `services/pmb-service`: **LULUS** (`go build ./...` exit 0).
- Architecture Boundaries (`arch_test.go`): **LULUS**.
- Pembersihan Direktori: 11 direktori service lama dan folder `_template` telah dihapus secara bersih.
