# Arsitektur Sistem PMB Backend (CR-011: 3-Service Consolidated Architecture)

Dokumen ini mendefinisikan arsitektur sistem PMB (Penerimaan Mahasiswa Baru) setelah pelaksanaan **CR-011 (Konsolidasi Layanan PMB)**. Arsitektur telah bertransformasi dari 13 microservices terdistribusi menjadi **3 layanan aplikasi berdaya guna tinggi** dan **2 basis data PostgreSQL**.

---

## 1. Topologi Layanan (3-Service Topology)

```
                       [ KLIEN / BROWSER ]
                                |
                                v
               +----------------------------------+
               |             GATEWAY              |
               |  (Port :8000 publik / :8081 int) |
               |                                  |
               |  - Reverse Proxy & Routing       |
               |  - Validasi JWT RSA In-Process   |
               |  - HMAC Signer X-Gateway-Signature|
               |  - Modul Auth (pmb_auth DB)      |
               +----------------+-----------------+
                                |
        +-----------------------+-----------------------+
        | (Proxy / API Call)                            | (Proxy / API Call)
        v                                               v
+-------------------------------+             +-------------------+
|          PMB-SERVICE          |             |    OCR-SERVICE    |
|   (Modular Monolith :8080)    |             | (FastAPI :8089)   |
|                               |             +-------------------+
| 10 Modul Bisnis:              |
|  - organization  - identity   |
|  - notification  - admission  |
|  - finance       - exam       |
|  - scoring       - selection  |
|  - student       - nim        |
|                               |
| Basis Data: pmb_app (10 Skema)|
+-------------------------------+
```

### Rincian 3 Layanan:
1. **`gateway` (Go, Gin Engine)**
   - **Port Publik**: `:8000` (Routing downstream, CORS, Rate Limit, Auth login/register/refresh/OTP/audit).
   - **Port Internal**: `:8081` (Endpoint verifikasi token, user management inter-service).
   - **Auth Terintegrasi (In-Process)**: Seluruh autentikasi (`auth-service` lama) dikonsolidasikan langsung ke dalam Gateway. Validasi JWT dilakukan lokal via public key tanpa round-trip HTTP mandiri.
   - **Database**: `pmb_auth` (Users, Roles, Permissions, UserRoles, AuditLogs, RefreshTokens, OTPs).

2. **`pmb-service` (Go, Gin Engine, Modular Monolith)**
   - **Port**: `:8080`.
   - **Domain Bisnis**: Menggabungkan 10 service fungsional PMB ke dalam satu binary monolitik modular dengan batas (*package boundaries*) yang ketat.
   - **Schema Isolation**: Terhubung ke basis data tunggal `pmb_app` menggunakan 10 skema PostgreSQL terisolasi (`organization`, `identity`, `notification`, `admission`, `finance`, `exam`, `scoring`, `selection`, `student`, `nim`). Dilarang melakukan join lintas schema di level ORM/database.
   - **Background Job Runner**: Menggunakan `pg_try_advisory_lock` untuk memastikan eksekusi job periodik dan Transactional Outbox terdistribusi tetap aman dan eksklusif.

3. **`ocr-service` (Python, FastAPI, Tesseract / TrOCR)**
   - **Port**: `:8089`.
   - **Fungsi**: Ekstraksi OCR identitas KTP dan dokumen berkecepatan tinggi.

---

## 2. Topologi Basis Data (2-Database Architecture)

1. **`pmb_auth`**
   - Basis data khusus keamanan, akun pengguna, otorisasi RBAC, riwayat audit, dan kredensial token.
2. **`pmb_app`**
   - Basis data aplikasi PMB yang menaungi 10 skema independen:
     - `organization` (Fakultas, Program Studi, Profil Kampus)
     - `identity` (Biodata, Alamat, Orang Tua/Wali, Riwayat Pendidikan, KTP Scan)
     - `notification` (Log Notifikasi, Activity Events, Template Broadcast, Broadcast Logs)
     - `admission` (Periode, Jalur Pendaftaran, Kuota, Pendaftaran, Dokumen, Outbox)
     - `finance` (Skema Biaya, Tagihan, Pembayaran, Kuitansi, Angsuran, Refund)
     - `exam` (Sesi Ujian, Kartu Ujian, Bank Soal, CBT Attempts, CBT Answers)
     - `scoring` (Komponen Nilai, Nilai Peserta, Kunci Prodi)
     - `selection` (Rekomendasi Kelulusan, Pengumuman, Hasil Seleksi)
     - `student` (Pendaftaran Ulang, Induk Mahasiswa Baru, Gugus PKKMB, SIAKAD Export)
     - `nim` (Template Format NIM, Nomor Urut Sequence, Riwayat Penerbitan NIM)

---

## 3. Keamanan & Komunikasi Antar-Layanan
- **HMAC SHA-256 Signature**: Gateway menandatangani setiap request downstream yang diteruskan ke `pmb-service` menggunakan header `X-Gateway-Signature` dan menyertakan konteks identitas `X-User-ID`, `X-User-Roles`, `X-User-Permissions`.
- **Internal Token**: Komunikasi langsung inter-service dilindungi oleh header `X-Internal-Token`.
