# PRD Tambahan — Kesiapan Rilis PMB (CR-013), Referal & Komisi Guru BK (CR-014), Konsolidasi 3 Service (CR-011), Auth Dua Versi & Persiapan SSO (CR-012)

| | |
|---|---|
| **Produk** | Sistem PMB (backend) — repo `diaricakarsa/pmb-backend` |
| **Isi** | **Bagian R (CR-013):** aplikasi pendaftaran camaba stabil tanpa bug, payment gateway Duitku siap, jalur gratis tanpa pembayaran, data demo, bahan buku panduan. **Bagian F (CR-014):** kode referal guru BK — siswa membayar ke guru BK, guru BK memverifikasi sehingga pembayaran di sistem dilewati, guru BK menyetor ke kampus, dan **komisi** guru BK (persen per transaksi atau nominal per mahasiswa). **Bagian I (CR-011):** 13 service menjadi **3 service** tanpa mengubah API. **Bagian II (CR-012):** auth dua versi (Auth Kampus/SSO + Auth PMB) dengan skema `users` – `service_accounts` – `user_accounts`, persiapan SSO |
| **Sumber** | Arahan lead (pak Kemal) & pak Endar 3–4 Okt 2026: launching PMB **22 Okt** dengan 150 guru BK; halaman khusus guru BK seperti halaman referal sistem BTH sekarang; *"fokus di point 2 … kalau testnya udah okey kasih tau, mau kita bikin buku panduannya"*; *"minggu ini sistem untuk mahasiswanya udah fix, gak ada bug, termasuk sudah bisa payment gateway"*; sosialisasi **Selasa 6 Okt malam**; *"siswa bayar terlebih dahulu ke guru BK … masukkan kode referal … di halaman guru BK diverifikasi, pembayarannya otomatis di-bypass … guru BK setelah uangnya dikumpulkan disetor ke pihak kampus"*; *"akun guru BK dibuat oleh admin kampus, ada settingan berapa persen komisinya per transaksi"*; *"bisa juga nilai tertentu (mis. 200rb/100rb) per mahasiswa yang daftar"*; *"yang gratis atau jalur tertentu tidak ada komisi, harus ada settingannya"*; *"biaya pendaftaran free: tidak usah ke payment, langsung ke step selanjutnya"*; pengaturan komisi & cara penyelesaiannya seluruhnya di pengaturan admin; pemangkasan service dikerjakan **langsung sekarang** (tidak ditunda); pemangkasan service jadi 3, auth 2 versi, persiapan SSO, skema *UserTabel, ServiceAccount, UserAccount* |
| **Versi** | v5 — 4 Oktober 2026 (satu dokumen untuk CR-011..CR-014; **tidak ada file prompt terpisah**; pemangkasan service dikerjakan langsung di `main` sebelum fitur referal) |
| **Prasyarat** | CR-001..CR-010 selesai dan `final-check` AMAN |
| **Hubungan dengan dokumen lain** | Aturan umum tetap `PRD-LENGKAP.md`, `PRD-TAMBAHAN-CR006-008.md`, `PRD-TAMBAHAN-CR009-010.md`. Bagian R & F menambah/memperbaiki fitur tanpa mengubah kontrak lama kecuali yang ditandai **[UBAH]**. Bagian I hanya mengubah susunan deployment & struktur kode. Bagian II menambah struktur data & endpoint auth tanpa mengubah perilaku endpoint PMB. Aturan lama tentang "satu service per domain", "database per service", atau "HTTP internal antar service" digantikan Bagian I setelah Bagian I dikerjakan. |

---

## 0. CARA KERJA AI AGENT — baca sampai habis sebelum mulai

**Dokumen ini adalah satu-satunya instruksi kerja.** Tidak ada prompt lanjutan. Kerjakan seluruh fase di bawah secara berurutan dan mandiri, tanpa menunggu persetujuan di antara langkah, kecuali kondisi berhenti di 0.7.

### 0.1 Pemeriksaan awal (sekali)
1. `git checkout main && git pull --rebase origin main`.
2. Bila di `main` sudah ada pekerjaan konsolidasi (folder `services/pmb-service`, folder `services/gateway`, atau commit berlabel CR-011/CR-012 selain dokumen), **berhenti** dan laporkan daftar commit tersebut. Bila tidak ada, lanjut.

### 0.2 Urutan fase & branch

| Fase | Isi | Target | Branch | Tag |
|---|---|---|---|---|
| **R** | Bagian R (CR-013) | Senin 5 Okt malam (paling lambat Selasa 6 Okt siang) | `main` | `v0.5.1-rilis` |
| **K** | Bagian I lalu Bagian II (CR-011, CR-012) — pemangkasan 3 service + auth dua versi/SSO | Selasa 6 – Sabtu 10 Okt | `main` | `v0.6.0-3-services`, `v0.7.0-sso-ready` |
| **F** | Bagian F (CR-014) — dibangun langsung sebagai modul `referral` di `pmb-service` | Minggu 11 – Jumat 16 Okt | `main` | `v0.8.0-referal` |
| Freeze | Hanya perbaikan bug P0/P1 | 19–21 Okt | `main` | `v0.8.x` (patch) |

- Semua fase di branch `main`, berurutan R → K → F. Fitur referal dibangun **setelah** pemangkasan supaya langsung berada di arsitektur akhir (tidak dipindah dua kali).
- Demo Selasa 6 Okt dijalankan dari tag `v0.5.1-rilis` (23.6), sehingga pekerjaan fase K di `main` tidak mengganggu demo.
- Langkah rinci tiap fase ada di **Bagian 7** (milestone R0–R3, M0–M8, F1–F5).

### 0.3 Aturan setiap langkah (milestone)
1. Awal langkah: `git pull`; baca `docs/BUGS.md` di `main` — bila ada bug baru berstatus `Open` dengan severity P0/P1, **kerjakan dulu di `main`** (lihat 0.5).
2. Akhir langkah: `go build/vet/test` semua module + `pytest` OCR → `docker compose -f deploy/docker-compose.yml down -v` → `up -d --build` → semua container healthy → `scripts/e2e.sh` penuh `FAIL: 0` (`PAYMENT_PROVIDER=mock`) → mulai fase R selesai: `scripts/smoke-camaba.sh` lulus → commit → push.
3. Skenario e2e baru **ditambahkan di akhir** `scripts/e2e.sh`; skenario lama tidak diubah kecuali ditandai [UBAH]. Ringkasan tetap `Total PASS: X, FAIL: Y`.
4. Akhir setiap fase: jalankan `powershell -ExecutionPolicy Bypass -File scripts/final-check.ps1 -Tahap <rilis|konsolidasi|penuh>` (R = `rilis`, K = `konsolidasi`, F = `penuh`) (bila PowerShell tersedia), perbaiki sampai AMAN, buat tag, push tag, tulis laporan fase (0.8), lalu **langsung lanjut** ke fase berikutnya.

### 0.4 Larangan
- Mengubah isi PRD lain atau logika `scripts/final-check.ps1`.
- Force push; meng-commit `.env`, kredensial, private key, data pribadi asli, atau tangkapan layar sistem lama. Semua contoh data memakai data rekaan.
- Menulis sendiri kriptografi/protokol OpenID Connect (Bagian II wajib memakai library matang).

### 0.5 Bug dari tim
Tim (Diar/Wahhed) menambahkan bug ke `docs/BUGS.md` di `main` (format 21.1). Setiap awal langkah periksa bug baru: P0/P1 dikerjakan lebih dulu (commit + push + e2e), lalu lanjutkan langkah yang tertunda. Bug yang ditemukan di versi demo (`v0.5.1-rilis`) dan masih ada di `main` diperbaiki di `main`. Selama freeze 19–21 Okt, `main` hanya menerima perbaikan P0/P1.

### 0.6 Pencatatan
Catat CR-013, CR-014, CR-011, CR-012 di `docs/CHANGE_REQUESTS.md` (open → done) dan seluruh keputusan Bagian 8 di `docs/DECISIONS.md`.

### 0.7 Kondisi berhenti
Berhenti, tulis alasannya di `docs/LAPORAN-AGENT.md`, commit, push, dan tunggu instruksi bila:
- e2e/smoke gagal dan tidak bisa diperbaiki setelah 3 kali percobaan;
- perilaku API atau isi token PMB terpaksa berubah di luar yang ditandai [UBAH];
- migrasi data berisiko kehilangan data, atau ditemukan data staging/produksi yang harus dipertahankan;
- dokumen ini bertentangan dengan dirinya sendiri.

### 0.8 Laporan
Setiap fase selesai, tambahkan bagian baru di `docs/LAPORAN-AGENT.md` (commit + push) dan tampilkan ringkasannya di chat: status tiap milestone, bug ditemukan/diperbaiki/sisa, hasil e2e & smoke, hasil final-check, endpoint baru/berubah untuk frontend, variabel env baru yang harus diisi tim (nama variabel saja, tanpa nilai), hash commit terakhir, tag.

---

# BAGIAN R — CR-013 KESIAPAN RILIS APLIKASI PENDAFTARAN

Tujuan: aplikasi pendaftaran camaba **bisa didemokan Selasa 6 Okt malam** ke bagian promosi & marketing, akademik, dan tim IT BTH, **tanpa bug**, dengan **pembayaran sungguhan lewat Duitku** (kredensial sandbox & produksi, SMTP, dan server sudah disiapkan tim di `.env`; agent tidak membaca, mencetak, atau menyalin nilainya), lalu menjadi dasar **buku panduan**.

## 20. Lingkup alur yang harus bebas bug

Alur camaba dari awal sampai akhir, lewat gateway, sesuai kontrak `docs/contracts/`:
1. Daftar akun → OTP email → verifikasi → login (termasuk lupa/reset kata sandi, ganti email saat pending).
2. Dashboard 6 tahap, agenda, notifikasi lonceng.
3. Biodata (termasuk pindai KTP/OCR dan isian manual bila OCR gagal), pas foto.
4. Pilih jalur & prodi (pilihan kedua opsional), batalkan pilihan.
5. Berkas: unggah 5 persyaratan, revisi.
6. **Pembayaran formulir lewat Duitku** (VA/QRIS/transfer sesuai metode aktif), kuitansi, kedaluwarsa & perpanjang tagihan, konfirmasi manual oleh keuangan; **jalur gratis tanpa pembayaran** (26).
7. CBT (jadwal, mengerjakan, auto-submit), hasil, pengumuman, LoA.
8. Daftar ulang, UKT, NIM, aktivasi.
9. Riwayat & pindah jalur, pengunduran diri (CR-006..008).
10. Sisi admin yang dipakai saat demo: verifikasi berkas, rekonsiliasi & konfirmasi lunas, penjadwalan CBT, seleksi & publikasi, broadcast.

## 21. Pencarian & perbaikan bug

### 21.1 Catatan bug `docs/BUGS.md`
Satu tabel: `ID | Tanggal | Sumber (agent/Diar/Wahhed/lead) | Halaman/Endpoint | Langkah | Hasil | Harapan | Severity | Status | Commit`.
- Severity: **P0** (alur terhenti/data salah/uang salah), **P1** (fungsi utama salah tetapi ada jalan lain), **P2** (tampilan/pesan kurang tepat), **P3** (kosmetik/usulan).
- Status: `Open`, `Fixed`, `Wontfix` (wajib alasan), `NeedInfo`.
- **Rilis hanya boleh bila tidak ada P0/P1 berstatus Open** (diperiksa `final-check`).

### 21.2 Audit mandiri (wajib, temuan masuk BUGS.md)
1. Jalankan e2e penuh dengan log semua container; **setiap respons 5xx, panic, atau error log** selama e2e dijelaskan atau diperbaiki.
2. Uji kontrak: setiap endpoint yang dipakai halaman camaba dibandingkan dengan `docs/contracts/*.schema.json` (field hilang, tipe salah, null tak terduga).
3. Kasus tepi: klik ganda/kirim ulang (idempotensi), callback Duitku ganda/terlambat/urutan terbalik, tagihan kedaluwarsa lalu dibayar, unggah file terlalu besar/format salah, sesi habis di tengah CBT, zona waktu batas waktu (WIB), paginasi kosong.
4. **Semua pesan error ke pengguna berbahasa Indonesia** dan bebas detail teknis (tidak ada `EOF`, `sql:`, stack trace, pesan Inggris mentah).
5. CORS & cookie untuk domain frontend demo/produksi (`CORS_ALLOWED_ORIGINS`).
6. Perbaiki semua P0/P1; P2 bila sempat sebelum target fase R.

### 21.3 Smoke test `scripts/smoke-camaba.sh`
Skrip singkat (≤ 3 menit) yang menjalankan alur 1–8 di Bagian 20 (termasuk jalur berbayar dan jalur gratis) dengan `PAYMENT_PROVIDER=mock` lewat gateway, mencetak `SMOKE PASS/FAIL` per langkah.

## 22. Payment gateway Duitku siap pakai

1. **Mode lingkungan** `DUITKU_ENV=sandbox|production` memilih URL; tidak ada URL yang di-hardcode:

   | Keperluan | Sandbox | Produksi |
   |---|---|---|
   | Create invoice (POP) | `https://api-sandbox.duitku.com/api/merchant/createInvoice` | `https://api-prod.duitku.com/api/merchant/createInvoice` |
   | Library POP JS (frontend) | `https://app-sandbox.duitku.com/lib/js/duitku.js` | `https://app-prod.duitku.com/lib/js/duitku.js` |
   | Cek status & daftar metode (WebAPI) | `https://sandbox.duitku.com/webapi/...` | `https://passport.duitku.com/webapi/...` (**verifikasi ke docs.duitku.com**, catat sumbernya) |

   `GET /finance/payment-config` (publik, tanpa rahasia) mengembalikan `env` dan URL library POP untuk frontend.
2. **Pengaman konfigurasi saat start:** `APP_ENV=production` mewajibkan `PAYMENT_PROVIDER=duitku`; `DUITKU_ENV=sandbox` di produksi hanya bila `ALLOW_SANDBOX_IN_PRODUCTION=true` (server demo). Merchant code/API key kosong → service menolak start dengan pesan jelas. API key tidak pernah ditulis ke log.
3. **Callback:** verifikasi signature + konfirmasi `transactionStatus` sebelum menandai lunas; idempoten; nominal harus sama dengan tagihan; callback untuk tagihan yang sudah lunas lewat jalur lain dicatat dan diabaikan; balasan sesuai dokumentasi Duitku.
4. **Rekonsiliasi berkala:** job memeriksa tagihan `pending` ke `transactionStatus` (default tiap 5 menit, maks. 24 jam setelah dibuat).
5. **Return URL** ke halaman frontend `.../pembayaran/selesai?invoice=...`; lunas hanya dari callback/rekonsiliasi, tidak dari return URL.
6. **Batas nominal** minimum per metode (dari `getpaymentmethod`) dihormati; metode yang tidak mendukung nominal disembunyikan.
7. `scripts/duitku-sandbox-check.ps1` ditambah `-Env production` yang **hanya** memanggil daftar metode pembayaran (tanpa membuat tagihan).
8. `docs/DUITKU-PRODUKSI.md`: langkah pindah sandbox → produksi, kebutuhan (merchant code & API key produksi, proyek/website disetujui di dashboard Duitku, URL callback HTTPS publik, URL return), cara uji transaksi nominal kecil, cara kembali ke sandbox.

## 23. Kesiapan demo & server
1. `deploy/docker-compose.prod.yml` untuk **arsitektur sekarang**: `APP_ENV=production`, tanpa Mailpit (SMTP lewat env), hanya gateway yang di-publish, `restart: unless-stopped`, volume data, healthcheck. (Ditulis ulang di Bagian I M4.)
2. `docs/DEPLOY.md`: (a) server/VPS dengan domain & HTTPS (reverse proxy), (b) laptop + `cloudflared` tunnel untuk callback Duitku saat demo; daftar env wajib; backup & restore Postgres (`make backup`, `make restore`).
3. **Data demo:** `make demo-seed` (profil institusi contoh, fakultas & prodi, jalur berbayar **dan jalur gratis**, gelombang aktif dengan tanggal relatif hari ini, jadwal CBT, bank soal contoh, beberapa camaba rekaan di berbagai tahap; setelah fase F juga guru BK & kode referal contoh) dan `make demo-reset`. Akun demo per role dari env `DEMO_PASSWORD`, **tidak** di-commit.
4. **Uji beban ringan** (k6 lewat Docker, `scripts/load/camaba.js`): 150 pengguna virtual 5 menit (login, dashboard, opsi jalur, notifikasi); target p95 < 800 ms, error < 1%; hasil di `docs/RELEASE-CHECKLIST.md`.
5. `docs/RELEASE-CHECKLIST.md`: env, CORS, SMTP, Duitku, backup, zona waktu `Asia/Jakarta`, rate limit, log, akun demo, hasil smoke & uji beban.
6. **Lingkungan demo dari tag:** `docs/DEPLOY.md` memuat cara menjalankan demo dari tag `v0.5.1-rilis` (`git checkout v0.5.1-rilis` → `make demo-reset`), dan cara memperbarui demo bila ada patch (`v0.5.2`, dst.) tanpa mengambil pekerjaan fase K yang belum selesai.

## 24. Bahan buku panduan `docs/PANDUAN-ALUR.md`
Per peran (**camaba**, **guru BK** setelah fase F, **admin PMB**, **keuangan**): urutan langkah, tujuan tiap langkah, status/label yang muncul, notifikasi, batas waktu, pesan error umum beserta solusinya, FAQ (lupa kata sandi, OTP tidak masuk, pembayaran belum terkonfirmasi, OCR gagal, jalur gratis, bayar lewat guru BK). Bahasa Indonesia sederhana tanpa istilah teknis; nama menu mengikuti frontend (`docs/FE-BE-GAP.md`).

## 25. Selesai fase R
- e2e lama lulus; skenario tambahan: callback ganda, callback nominal salah, rekonsiliasi saat callback hilang (server Duitku palsu), pengaman konfigurasi produksi (unit test), pesan error Indonesia untuk error code utama, **jalur gratis** (26).
- `scripts/smoke-camaba.sh` lulus; `docs/BUGS.md` tanpa P0/P1 Open; `docs/DUITKU-PRODUKSI.md`, `docs/DEPLOY.md`, `docs/RELEASE-CHECKLIST.md`, `docs/PANDUAN-ALUR.md` ada.
- CR-013 `done`; tag `v0.5.1-rilis`; `final-check -Tahap rilis` AMAN.

## 26. Jalur dengan biaya pendaftaran gratis [UBAH]
1. Master jalur boleh `registration_fee = 0`; `GET /admission/options` menampilkan `registration_fee_label = "Gratis"`.
2. Saat camaba menyimpan pilihan jalur gratis, Finance membuat tagihan Rp 0 yang **langsung lunas** dengan metode `free` (tanpa Duitku, tanpa kuitansi pembayaran) dan mengirim event lunas yang sama, sehingga tahap berikutnya (verifikasi berkas, kelayakan CBT) terbuka otomatis. Camaba **tidak** diarahkan ke halaman pembayaran.
3. Dashboard: tahap "Biaya" berstatus selesai dengan label "Gratis — tidak perlu pembayaran"; `current_action` langsung ke langkah berikutnya. Endpoint pembayaran untuk tagihan gratis → `409 INVOICE_FREE`.
4. Pembatalan (CR-002): pendaftaran jalur gratis diperlakukan seperti belum membayar (boleh dibatalkan). Pindah jalur (CR-006) dari jalur gratis ke jalur berbayar menghasilkan tagihan sesuai biaya jalur tujuan.
5. Laporan & rekonsiliasi: tagihan `free` dihitung sebagai "Gratis", tidak masuk kas.

---

# BAGIAN F — CR-014 KODE REFERAL, PEMBAYARAN LEWAT GURU BK & KOMISI

## 30. Latar belakang & alur yang ditetapkan lead

Launching 22 Okt mengundang **150 guru BK**. Halaman guru BK dibuat mirip halaman **"Detail Kode Referal"** di sistem pendaftaran BTH sekarang: Informasi Referal (institusi, kode referal, penanggung jawab, kode pembayaran siswa, deskripsi, status), Total Pengguna, kartu Transaksi Pendaftaran (**All Transaksi (Success)**, **Tagihan Saya ke Kampus**, **Sudah Disetorkan ke Kampus**, **Antrian Verifikasi Pembayaran**), dan **Daftar Pengguna Kode Referal** (pencarian nama/email; kolom `# | Nama | Email | NIK | NISN | Nominal | Tipe | Status Approve | Keuangan Kampus`; paginasi). Satu penanggung jawab dapat memegang beberapa kode.

Alur yang ditetapkan lead:
1. **Siswa membayar biaya pendaftaran terlebih dahulu ke guru BK** (di luar sistem).
2. Siswa **memasukkan kode referal** guru BK-nya di aplikasi.
3. Di halaman guru BK, pembayaran siswa itu **diverifikasi**; setelah diverifikasi, **pembayaran di sistem otomatis dilewati** (tagihan formulir dianggap lunas) dan siswa lanjut ke langkah berikutnya.
4. Guru BK **menyetor uang yang terkumpul ke kampus**; keuangan memverifikasi setoran.
5. Guru BK mendapat **komisi**: **persen per transaksi** atau **nominal tetap per mahasiswa yang daftar** (mis. Rp 100.000 / Rp 200.000), diatur admin saat membuat akun guru BK dan dapat diubah. **Jalur gratis dan jalur tertentu tidak mendapat komisi** (diatur per jalur).
6. **Akun guru BK dibuat oleh admin kampus.**

## 31. Istilah

| Istilah | Arti |
|---|---|
| **Mitra** | Institusi pemilik kode (umumnya sekolah; bisa juga program kerja sama/komunitas) |
| **Kode referal** | Kode unik yang dimasukkan siswa (mis. `SMKCONTOH01`) |
| **Penanggung jawab (PIC)** | Akun yang mengelola kode: **guru BK** (role `guru_bk`) atau staf kampus |
| **Kode pembayaran siswa** | Kode referensi pembayaran siswa ke guru BK; default sama dengan kode referal |
| **Verifikasi** | Guru BK menyatakan siswa sudah membayar kepadanya → tagihan formulir di sistem lunas |
| **Setoran** | Penyerahan uang terkumpul dari guru BK ke kampus, dengan bukti, diverifikasi keuangan |
| **Komisi** | Bagian untuk guru BK per transaksi terverifikasi: `persen` atau `nominal` |

## 32. Data
Modul **`referral`** di **`pmb-service`** (dibangun setelah Bagian I), schema Postgres `referral` sendiri, mengikuti aturan batas modul 3.2: akses ke data pendaftar, tagihan, dan biodata hanya lewat facade modul `admission`, `finance`, `identity`.

| Tabel | Kolom utama |
|---|---|
| `referral_codes` | `id`, `code` (unik, A–Z & 0–9, 4–20 karakter, input tidak peka huruf besar/kecil), `payment_code` (unik, default = `code`), `institution_name`, `institution_type` (`sekolah`/`program`/`komunitas`), `npsn` (opsional), `description` (Markdown, disanitasi), `status` (`active`/`inactive`), `valid_from`, `valid_until` (opsional), `period_id` (opsional), **`commission_type`** (`persen`/`nominal`/`tidak_ada`), **`commission_value`** (persen 0–100 dua desimal, atau rupiah), **`commission_settlement_mode`** (`terpisah`/`potong_setoran`/kosong = ikut pengaturan umum), `created_by`, timestamps |
| `referral_code_pics` | `code_id`, `user_id`, `is_primary` — satu kode boleh >1 PIC, satu PIC boleh banyak kode |
| `referral_usages` | `id`, `code_id`, `application_id`, `user_id` (camaba), `consent_at`, `attached_at`, `locked_at`, `changed_by`, `change_reason` — satu pendaftaran maksimal satu kode |
| `referral_transactions` | `id`, `code_id`, `usage_id`, `invoice_id`, `type` (`REGISTRATION`; `REREGISTRATION` disiapkan), `amount`, `approve_status` (`PENDING`/`SUCCESS`/`REJECTED`), `campus_status` (`TAGIHAN`/`MENUNGGU_VERIFIKASI`/`DISETOR`), `approved_by`, `approved_at`, `reject_reason`, `settlement_id`, **`commission_type_snapshot`**, **`commission_value_snapshot`**, **`commission_amount`**, **`commission_status`** (`TIDAK_ADA`/`BELUM_DIBAYAR`/`DIBAYAR`/`DIPOTONG`), `commission_payout_id` |
| `referral_settlements` | `id`, `code_id`, `gross_amount`, `commission_deducted`, `amount` (yang disetor), `transfer_date`, `bank_name`, `reference_no` (unik), `proof_object_key` (MinIO privat), `notes`, `status` (`SUBMITTED`/`VERIFIED`/`REJECTED`), `submitted_by`, `verified_by`, `verified_at`, `reject_reason` |
| `referral_commission_payouts` | `id`, `code_id`, `amount`, `transaction_ids`, `paid_at`, `method`, `reference_no`, `proof_object_key`, `paid_by` |
| `referral_settings` (1 baris) | `commission_settlement_mode` (`terpisah` default / `potong_setoran`), `default_commission_type`, `default_commission_value` |
| Master jalur (admission, kolom baru) | **`referral_commission_enabled`** (bool, default `true`); jalur dengan `registration_fee = 0` selalu tanpa komisi |

## 33. Aturan bisnis

### 33.1 Siswa memasukkan kode
1. Kode dimasukkan di dashboard atau halaman jalur sebelum tagihan formulir lunas. Frontend boleh mengisi otomatis dari tautan `?ref=KODE` (endpoint auth tidak berubah).
2. `GET /api/v1/referral/codes/{code}/check` (publik, rate limit 10/menit/IP) → `{valid, institution_name, pic_name}`; kode tidak dikenal/nonaktif/di luar masa berlaku → `valid=false` dengan respons seragam.
3. Memasang kode wajib persetujuan: *"Saya sudah membayar biaya pendaftaran kepada guru BK dan setuju nama, email, NISN, dan status pendaftaran saya dapat dilihat oleh guru BK pemilik kode ini."* (`consent=true`, disimpan `consent_at`).
4. Kode dapat diganti/dilepas selama transaksi belum `SUCCESS`; sesudahnya → `409 REFERRAL_LOCKED`. Admin dapat mengganti dengan alasan (audit log).

### 33.2 Verifikasi guru BK & pembayaran dilewati
1. Bila pendaftaran berkode memiliki tagihan formulir (dibuat saat jalur dipilih, atau sudah ada saat kode dipasang), tagihan ditandai **ditagihkan ke mitra** dan transaksi referal `PENDING` dibuat. Camaba melihat status *"Menunggu verifikasi pembayaran oleh guru BK {nama} ({institusi})"*; membuat pembayaran Duitku untuk tagihan itu → `409 INVOICE_BILLED_TO_PARTNER`.
2. Tagihan yang ditagihkan ke mitra **tidak kedaluwarsa** selama transaksi `PENDING`.
3. Guru BK **memverifikasi** (satu per satu atau beberapa sekaligus) → transaksi `SUCCESS`, `campus_status = TAGIHAN`; Finance menandai tagihan lunas dengan metode `referral_collective` (kuitansi: "Dibayar melalui guru BK {nama} — {institusi}"); event lunas yang sama dikirim sehingga camaba **langsung lanjut ke langkah berikutnya**. Komisi dihitung saat ini (33.3).
4. Guru BK **menolak** (alasan wajib, mis. "belum membayar") → `REJECTED`; tagihan kembali menjadi tanggungan camaba (bisa dibayar sendiri lewat Duitku atau kode dipasang ulang); camaba diberi notifikasi.
5. Keuangan dapat memverifikasi/menolak atas nama guru BK (permission `referral.admin.approve`).
6. **Jalur gratis:** tidak ada tagihan berbayar, tidak ada transaksi referal, tidak ada komisi; siswa tetap tercatat sebagai pengguna kode dengan status "GRATIS".
7. Pembayaran `referral_collective` tercatat sebagai **piutang mitra** di rekonsiliasi, bukan kas masuk. Kas masuk dicatat di buku besar saat setoran diverifikasi (sumber "Setoran Guru BK {kode}").
8. Siswa yang sudah membayar sendiri lewat Duitku lalu memasang kode: kode dicatat untuk pelacakan saja (tanpa transaksi referal & tanpa komisi).

### 33.3 Komisi
1. Pengaturan per kode: `commission_type` `persen` (komisi = `commission_value`% × nominal transaksi) atau `nominal` (komisi = `commission_value` rupiah per mahasiswa terverifikasi) atau `tidak_ada`. Diisi admin di form **pembuatan akun guru BK** (34.2) dan dapat diubah kapan saja; nilai awal dari `referral_settings`.
2. Komisi hanya untuk transaksi `SUCCESS` pada jalur dengan `referral_commission_enabled = true` dan biaya > 0. Selain itu `commission_status = TIDAK_ADA`.
3. Tipe & nilai komisi **disalin (snapshot)** ke transaksi saat terverifikasi; perubahan pengaturan hanya berlaku untuk transaksi berikutnya. Pembulatan ke rupiah terdekat. Komisi `nominal` tidak boleh melebihi nominal transaksi (dibatasi).
4. Pengunduran diri/pembatalan setelah `SUCCESS` tidak mengubah komisi (biaya pendaftaran tidak dikembalikan).
5. Cara penyelesaian komisi diatur admin di **Pengaturan Referal** (`referral_settings.commission_settlement_mode`, berlaku umum) dan dapat ditimpa per kode (`referral_codes.commission_settlement_mode`). Perubahan berlaku untuk setoran berikutnya:
   - **`terpisah` (default):** guru BK menyetor penuh; keuangan membayar komisi terpisah dan mencatat pembayaran komisi (`referral_commission_payouts`, bukti opsional) → `commission_status = DIBAYAR`.
   - **`potong_setoran`:** guru BK menyetor nominal dikurangi komisi; saat setoran diverifikasi, komisi transaksi tersebut menjadi `DIPOTONG`.

### 33.4 Setoran guru BK ke kampus
1. Guru BK memilih transaksi `TAGIHAN` (atau "semua") → sistem menghitung `gross_amount`, `commission_deducted` (0 pada mode `terpisah`), `amount`; guru BK mengisi tanggal transfer, bank, nomor referensi (unik), unggah bukti (PDF/JPG/PNG ≤ 5 MB) → setoran `SUBMITTED`; transaksi → `MENUNGGU_VERIFIKASI`.
2. Keuangan **verifikasi** → `VERIFIED`, transaksi `DISETOR`, entri buku besar; **tolak** (alasan wajib) → transaksi kembali `TAGIHAN`.

### 33.5 Ringkasan per kode (dengan `updated_at`)
| Kartu | Rumus |
|---|---|
| Total Pengguna | jumlah pendaftaran yang memakai kode |
| All Transaksi (Success) | Σ nominal transaksi `SUCCESS` |
| Tagihan Saya ke Kampus | Σ (nominal − komisi bila mode `potong_setoran`) transaksi `SUCCESS` yang `campus_status` ≠ `DISETOR` |
| Sudah Disetorkan ke Kampus | Σ `amount` setoran `VERIFIED` |
| Antrian Verifikasi Pembayaran | Σ nominal transaksi `PENDING` (siswa sudah memasukkan kode, belum diverifikasi guru BK) |
| Komisi Saya | total komisi, sudah dibayar/dipotong, belum dibayar |
Tambahan: `settlement_pending_amount` (Σ setoran `SUBMITTED`).

### 33.6 Privasi
- Guru BK hanya melihat pendaftar yang memakai **kodenya sendiri**; kode lain → `404`.
- NIK **disamarkan** untuk guru BK (`3204********0002`); NIK utuh hanya untuk admin. Email & NISN tampil. Dokumen, biodata lengkap, nilai, dan data orang tua **tidak** dapat diakses guru BK.
- Akses guru BK ke daftar & ekspor dicatat di audit log.

## 34. Akun guru BK & hak akses

### 34.1 Role & permission (matriks di `docs/RBAC.md`)
| Permission | guru_bk | admin_pmb | keuangan | superadmin |
|---|---|---|---|---|
| `referral.partner.read` (kode sendiri, daftar pengguna, ringkasan, komisi sendiri) | ✓ | | | |
| `referral.partner.approve` (verifikasi/tolak pembayaran siswa) | ✓ | | | |
| `referral.partner.settle` (ajukan setoran) | ✓ | | | |
| `referral.admin.manage` (kode, akun guru BK, pengaturan komisi per kode, impor) | | ✓ | | ✓ |
| `referral.admin.read` (semua kode & laporan) | | ✓ | ✓ | ✓ |
| `referral.admin.approve` (verifikasi/tolak atas nama guru BK) | | | ✓ | ✓ |
| `referral.settlement.verify` | | | ✓ | ✓ |
| `referral.commission.pay` (catat pembayaran komisi) | | | ✓ | ✓ |
| `referral.settings.manage` (mode penyelesaian komisi, nilai awal komisi) | | ✓ | | ✓ |

### 34.2 Pembuatan akun oleh admin kampus
1. Satu form: data guru BK (nama, email, no. HP), institusi (nama, jenis, NPSN), kode referal (kosong → dibuat otomatis dari nama institusi + 2 angka acak, unik), deskripsi, **pengaturan komisi** (tipe & nilai). Hasil: akun `guru_bk` lewat API admin auth yang ada + kode + tautan PIC + email **aktivasi** (alur reset kata sandi, judul "Aktivasi Akun Guru BK — PMB", berlaku 72 jam). Email yang sudah terdaftar cukup ditambah role & dihubungkan ke kode.
2. **Impor CSV** untuk banyak guru BK: kolom `nama, email, no_hp, nama_institusi, jenis_institusi, npsn, kode (opsional), tipe_komisi, nilai_komisi, kirim_aktivasi (ya/tidak)`. Laporan per baris (dibuat/diperbarui/gagal + alasan); baris gagal tidak membatalkan baris lain. Aktivasi dapat dikirim serentak kemudian (`POST .../pics/send-activation`).
3. Tidak ada pendaftaran mandiri guru BK.
4. Role `guru_bk` milik service account **`pmb-mitra`** (Bagian II); akun guru BK mendapat `user_accounts` ke `pmb-mitra`. Login memakai `POST /api/v1/auth/login` yang sama (portal guru BK mengirim `service=pmb-mitra`); menu dari `GET /auth/me/permissions` (CR-009).

## 35. API (route gateway baru `/api/v1/referral/*` → `pmb-service`)

| Peran | Endpoint |
|---|---|
| Publik | `GET /referral/codes/{code}/check` |
| Camaba | `GET /referral/me`, `PUT /referral/me` `{code, consent}`, `DELETE /referral/me` |
| Guru BK | `GET /referral/partner/codes` (kartu: institusi, kode, status, total pengguna) · `GET /referral/partner/codes/{id}` (informasi + ringkasan 33.5) · `GET /referral/partner/codes/{id}/users?q=&approve_status=&campus_status=&page=&per_page=` (kolom: no, nama, email, NIK tersamar, NISN, jalur, nominal, tipe, status approve, keuangan kampus, komisi, tanggal pakai kode, prodi pilihan, tahap pendaftaran) · `GET .../users/export?format=csv` · `POST /referral/partner/transactions/approve` `{ids[]}` · `POST /referral/partner/transactions/{id}/reject` · `GET/POST /referral/partner/codes/{id}/settlements` (multipart) · `GET /referral/partner/codes/{id}/commissions` |
| Admin | `GET/POST /referral/admin/codes`, `GET/PUT /referral/admin/codes/{id}`, `PATCH .../status`, `PUT .../pics`, `PUT .../commission` · `GET /referral/admin/codes/{id}/users` · `POST /referral/admin/pics` (form 34.2), `POST /referral/admin/pics/import` (CSV), `POST /referral/admin/pics/send-activation` · `POST /referral/admin/transactions/approve` `{ids[]}`, `POST .../{id}/reject` · `GET /referral/admin/settlements?status=`, `POST .../{id}/verify`, `POST .../{id}/reject` · `GET /referral/admin/commissions?status=&code_id=`, `POST /referral/admin/commission-payouts` (multipart) · `GET/PUT /referral/admin/settings` · `GET /referral/admin/report?period_id=` + `export` · `PUT /referral/admin/applications/{id}/referral` (ganti kode + alasan) |
| Master jalur | Endpoint master jalur yang ada menerima/mengembalikan field `referral_commission_enabled` |

Kontrak JSON di `docs/contracts/referral-*.schema.json`; label status Bahasa Indonesia disertakan (`approve_status_label`, `campus_status_label`, `commission_status_label`).

Integrasi antar modul di `pmb-service` lewat facade (tanpa HTTP): `finance` menyediakan `MarkBilledToPartner` / `ReleaseToApplicant`, `SettleByPartner`, `MarkFree`, dan pencatatan buku besar setoran; event `invoice.created` dan `invoice.paid` dari outbox `finance` (dispatcher in-process) dipakai modul `referral`; data pendaftar & biodata lewat facade `admission` dan `identity`. Tidak ada query lintas schema.

## 36. Notifikasi
- Guru BK: **ringkasan harian 07.00 WIB** (pengguna baru, pembayaran menunggu verifikasi) lewat email + in-app; setoran diverifikasi/ditolak; komisi dibayar.
- Camaba: kode terpasang; pembayaran diverifikasi/ditolak guru BK.
- Keuangan: setoran baru (in-app).
- Aktivitas tercatat di log aktivitas panitia (CR-009 2.9).

## 37. Halaman frontend (baris baru di `docs/FE-BE-GAP.md`)
Guru BK: **Daftar Kode Saya**, **Detail Kode Referal** (informasi, kartu ringkasan, pencarian, tabel, paginasi, ekspor, verifikasi/tolak, pilih beberapa), **Setoran** (form + riwayat), **Komisi Saya**. Camaba: isian kode + persetujuan + status "Menunggu verifikasi guru BK"; jalur gratis tanpa halaman pembayaran. Admin: **Guru BK & Kode Referal** (form pembuatan akun + komisi, impor CSV, kirim aktivasi), **Verifikasi Setoran**, **Komisi** (daftar & catat pembayaran), **Pengaturan Referal**, **Laporan Referal**, kolom "Komisi referal" di master jalur. Acuan tampilan: halaman referal sistem BTH sekarang.

## 38. Pengujian fase F
Unit test: rumus ringkasan (kedua mode), komisi persen/nominal/snapshot/batas, jalur tanpa komisi, penyamaran NIK, aturan kunci, kode otomatis, validasi CSV. Skenario e2e ditambahkan di akhir:
1. Admin membuat akun guru BK (komisi persen 10%) → email aktivasi (Mailpit) → guru BK menyetel kata sandi → login → melihat kodenya.
2. Cek kode valid/tidak valid; pasang kode tanpa `consent` → 422; dengan `consent` → muncul di daftar guru BK dengan NIK tersamar.
3. Siswa jalur berbayar memasang kode → tagihan ditagihkan ke mitra; bayar Duitku → `409 INVOICE_BILLED_TO_PARTNER`; "Antrian Verifikasi" bertambah.
4. Guru BK memverifikasi → tagihan lunas `referral_collective`, dashboard camaba langsung ke langkah berikutnya; "All Transaksi" & "Tagihan ke Kampus" bertambah; komisi = 10% nominal.
5. Ubah komisi kode menjadi nominal Rp 100.000 → transaksi lama tetap 10%, transaksi baru Rp 100.000.
6. Jalur dengan `referral_commission_enabled=false` → transaksi `SUCCESS` tanpa komisi; jalur gratis → status "GRATIS", tanpa transaksi.
7. Guru BK menolak → camaba dapat membayar sendiri lewat Duitku (mock).
8. Setoran mode `terpisah`: guru BK mengajukan dengan bukti → `MENUNGGU_VERIFIKASI`; keuangan verifikasi → `DISETOR`, buku besar bertambah; keuangan mencatat pembayaran komisi → `DIBAYAR`. Penolakan setoran mengembalikan ke `TAGIHAN`.
9. Pengaturan umum `potong_setoran`: nominal setoran = gross − komisi; setelah verifikasi komisi `DIPOTONG`. Kode dengan override `terpisah` tetap menyetor penuh.
10. Akses: guru BK A membuka kode guru BK B → 404; `guru_bk` ke endpoint admin → 403; camaba ke endpoint guru BK → 403.
11. Setelah `SUCCESS` camaba mengganti kode → `409 REFERRAL_LOCKED`; admin mengganti dengan alasan → berhasil + audit.
12. Impor CSV 3 baris (1 tidak valid) → 2 berhasil, 1 gagal dengan alasan.
13. Pencarian, paginasi, ekspor CSV (NIK tersamar); rate limit cek kode → 429.

## 39. Dokumentasi & selesai fase F
`docs/REFERRAL.md` (alur, komisi & mode penyelesaian, rumus ringkasan, privasi, format CSV), `docs/RBAC.md`, `docs/API.md` bagian "Referal, Guru BK & Komisi", Postman, kontrak, class diagram & ERD `pmb-service` (modul referral, admission, finance), `docs/PANDUAN-ALUR.md` bagian guru BK, `docs/FE-BE-GAP.md`. CR-014 `done`; tag `v0.8.0-referal`; `final-check -Tahap penuh` AMAN.

---

# BAGIAN I — CR-011 KONSOLIDASI 3 SERVICE

> Dikerjakan di `main` **langsung setelah fase R** (lihat 0.2). Skenario e2e fase R (termasuk jalur gratis) ikut menjadi baseline. Modul `referral` pada struktur di bawah dibuat baru di fase F.

---

## 1. Kondisi sekarang

13 service aplikasi + 4 container infrastruktur (± 17 container):

| Kelompok | Service |
|---|---|
| Pintu masuk | api-gateway |
| Akun & data dasar | auth, identity, organization |
| Alur PMB | admission, finance, exam, scoring, selection, student, nim |
| Pendukung | notification, ocr (Python) |
| Infrastruktur | postgres, redis, minio, mailpit (dev) |

Masalah: biaya server dan beban operasional (13 image, 13 proses, 13 healthcheck, 11 database, panggilan HTTP internal yang banyak).

## 2. Target: 3 service

| # | Service baru | Isi (service lama) | Alasan |
|---|---|---|---|
| 1 | **`gateway`** | api-gateway **+ auth** | Pintu masuk tunggal dan identitas (login, token, RBAC/permission, JWKS) disatukan. Kelak dapat dipakai bersama oleh sistem kampus lain (SIAKAD, dll.) yang dipasang di belakang gateway yang sama |
| 2 | **`pmb-service`** | identity, organization, notification, admission, finance, exam (termasuk CBT), scoring, selection, student, nim | Seluruh logika bisnis PMB sebagai **modular monolith**: satu proses, satu database, modul tetap terpisah di dalam kode |
| 3 | **`ocr-service`** | ocr (Python) | Bahasa dan dependensi berbeda (PaddleOCR/OpenCV). Dibuat **opsional**: dapat dimatikan di luar periode pendaftaran untuk menghemat resource |

Infrastruktur tetap: postgres (1 instance), redis, minio, mailpit (dev saja).

**Target hasil:** 3 container aplikasi (2 bila OCR dimatikan), 2 database (`pmb_auth` untuk gateway, `pmb_app` untuk pmb-service).

## 3. Arsitektur `pmb-service` (modular monolith)

### 3.1 Struktur kode
```
services/
  gateway/                 # dari api-gateway + auth-service
    cmd/gateway/main.go
    internal/proxy/ ...    # kode gateway lama
    internal/auth/ ...     # kode auth lama (model, repository, service, controller, token, mailer, seeder)
    migrations/            # migration auth lama, database pmb_auth
  pmb-service/
    cmd/pmb/main.go
    internal/platform/     # config, db, redis, storage (MinIO), outbox, job runner, http server,
                           # gatewayauth, rbac, internal client ke gateway & ocr
    internal/modules/
      identity/  organization/  notification/  admission/  referral/  finance/
      exam/  scoring/  selection/  student/  nim/
        api.go             # FACADE publik modul (interface + DTO) — satu-satunya pintu antar modul
        internal/...       # model, repository, service, controller, dto (pindahan dari service lama)
        migrations/        # migration modul, schema Postgres sendiri
  ocr-service/             # tidak berubah
pkg/                       # shared package tetap
```

### 3.2 Aturan batas modul (wajib)
1. Setiap modul memiliki **schema Postgres sendiri** di database `pmb_app` (`identity`, `organization`, `notification`, `admission`, `referral`, `finance`, `exam`, `scoring`, `selection`, `student`, `nim`). Migration per modul dijalankan berurutan saat start (golang-migrate dengan tabel versi per schema).
2. Modul **hanya** boleh membaca/menulis tabel di schema-nya sendiri. **Dilarang JOIN atau query lintas schema.**
3. Modul lain hanya boleh dipanggil lewat **facade** `modules/<nama>/api.go` (interface Go). Dilarang mengimpor paket `internal` milik modul lain. Ditegakkan oleh **arch test** (Go test yang memeriksa import graph dengan `go list -deps`) dan `depguard` di golangci-lint; pelanggaran membuat test gagal.
4. Panggilan **sinkron** lintas modul yang dulu lewat `/internal/*` HTTP menjadi **pemanggilan fungsi facade** (tanpa HTTP, tanpa `X-Internal-Token`).
5. Perubahan **asinkron** lintas modul (mis. pembayaran lunas → status pendaftaran, notifikasi, outbox CR-002/006/007) tetap memakai **outbox per modul**, tetapi dispatcher-nya **in-process** (memanggil handler facade modul tujuan, bukan HTTP). Idempotency key tetap dipertahankan.
6. Satu transaksi DB **tidak boleh** mencakup tabel lebih dari satu modul. (Supaya modul tetap bisa dipisah lagi kelak.)
7. Konfigurasi per modul memakai prefix env (`FINANCE_…`, `EXAM_…`, dst.); konfigurasi bersama di `platform`.

### 3.3 HTTP di dalam `pmb-service`
- Satu server HTTP. Setiap modul mendaftarkan router-nya di **prefix path yang sama dengan upstream lama**, sehingga `routes.yaml` gateway cukup mengganti URL tujuan.
- Middleware `pkg/gatewayauth` (signature gateway) dan `pkg/rbac` tetap dipasang untuk semua route kecuali `/health*` dan `/internal/*`.
- `/internal/*` yang **masih diperlukan** hanya untuk dipanggil gateway (mis. validasi scope prodi/fakultas ke modul organization) dan dilindungi `X-Internal-Token`, tidak diekspos keluar.
- `/health` (hidup) dan `/health/ready` (cek DB, Redis, MinIO, status setiap modul; OCR dilaporkan terpisah dan tidak membuat ready gagal bila `OCR_ENABLED=false`).

### 3.4 Job & worker latar belakang
Semua job yang dulu tersebar (outbox dispatcher, rekonsiliasi Duitku, reminder cicilan, kedaluwarsa tagihan, hapus akun pending, auto-submit CBT, antrean broadcast, retry notifikasi) berjalan di `pmb-service` lewat **satu job runner** dengan **Postgres advisory lock** per job (`pg_try_advisory_lock`), sehingga aman bila `pmb-service` dijalankan lebih dari satu replika.

## 4. `gateway` (api-gateway + auth)

1. Kode auth dipindah ke `services/gateway/internal/auth`. Database tetap `pmb_auth` (tidak berubah).
2. Validasi JWT **lokal** (kunci publik dari kunci privat yang sama; tidak perlu memanggil JWKS sendiri). Endpoint `/.well-known/jwks.json` tetap dipublikasikan untuk sistem lain.
3. Denylist, rotasi refresh token, rate limit, lockout, CORS, pembersihan header palsu, injeksi `X-User-*` dan `X-Gateway-Signature` ke upstream **tetap sama**.
4. Route `/api/v1/auth/*` ditangani langsung oleh modul auth di dalam gateway (tanpa proxy). Semua prefix lain (`/api/v1/identity`, `/organization`, `/admission`, `/finance`, `/exam`, `/scoring`, `/selection`, `/student`, `/nim`, `/notifications`, `/reports`, webhook Duitku) di-proxy ke `pmb-service`.
5. Endpoint internal auth yang dipakai pmb-service (tambah/cabut role, verifikasi password, batch user, dsb.) tetap tersedia di `gateway` pada **port internal terpisah** (mis. `:8081`, tidak di-publish ke host), dilindungi `X-Internal-Token`. `pmb-service` memanggilnya lewat klien internal di `platform`.
6. Gateway memanggil `pmb-service` `/internal/...` untuk validasi scope (organization), menggantikan panggilan ke organization-service lama.

## 5. Infrastruktur & biaya

1. Dua file compose:
   - `deploy/docker-compose.yml` (**development & pengujian**, dipakai `e2e.sh` dan `final-check.ps1`): `gateway`, `pmb-service`, `ocr-service`, `postgres`, `redis`, `minio`, `mailpit` — **semua menyala tanpa profile**, supaya e2e (termasuk OTP via Mailpit dan skenario OCR) tetap berjalan.
   - `deploy/docker-compose.prod.yml` (**server**): `gateway`, `pmb-service`, `postgres`, `redis`, `minio`; `ocr-service` hanya dengan `--profile ocr`; tanpa Mailpit (SMTP sungguhan). Tidak ada port selain gateway yang di-publish.
2. Init database: `pmb_auth` dan `pmb_app` (dengan schema per modul dibuat oleh migration). Database lama per service **tidak dipakai lagi**.
3. **OCR opsional (di server):** `OCR_ENABLED=false` → endpoint scan KTP membalas `503 OCR_DISABLED` dengan pesan "Fitur pindai KTP sedang tidak aktif, silakan isi biodata manual", dan field `ocr_available=false` ditambahkan di respons biodata (penambahan field, bukan perubahan). Di compose development OCR selalu menyala; perilaku `OCR_DISABLED` diuji lewat unit/integration test.
4. Batas resource di compose (`mem_limit`, `cpus`) untuk setiap container, dengan nilai awal dari pengukuran (Bagian 6.4).
5. Data: sistem belum dipakai produksi (konsolidasi selesai sebelum launching), jadi **tidak ada migrasi data antar database**; development cukup `down -v`. Bila agent menemukan data staging yang harus dipertahankan, **berhenti dan laporkan** sebelum menghapus database lama.

## 6. Pengujian & bukti

### 6.1 Wajib lulus di setiap langkah
- `go build`, `go vet`, `go test` semua module; `pytest` OCR.
- `docker compose down -v` → `up -d --build` → semua container healthy.
- `scripts/e2e.sh` penuh → `Total PASS: X, FAIL: 0` dengan **X tidak berkurang** dari baseline (kecuali penghapusan uji port lama di 6.3, dicatat jumlahnya).
- Kontrak `docs/contracts/*.schema.json` tetap valid.

### 6.2 Test baru
- **Arch test** batas modul (3.2 butir 2–3, 6): gagal bila ada import `internal` lintas modul atau query ke schema lain (pindai string SQL/`TableName()` di repository).
- Test job runner: dua instance bersamaan tidak menjalankan job yang sama (advisory lock).
- Test `OCR_ENABLED=false` → `503 OCR_DISABLED`.
- e2e tambahan (ditambahkan **di akhir** konsolidasi, tidak mengubah skenario lama): request langsung ke port `pmb-service` dengan token valid → 401 `GATEWAY_SIGNATURE_INVALID`; port internal gateway tidak dapat diakses dari host.

### 6.3 Pengujian yang boleh disesuaikan
Hanya skenario e2e yang memanggil **port service lama secara langsung** (mis. uji bypass ke `auth-service:8080`) yang boleh diganti dengan padanannya di service baru. Daftar penggantian dicatat di laporan dan `docs/DECISIONS.md`.

### 6.4 Laporan konsolidasi (`docs/CONSOLIDATION-REPORT.md`)
Ukur **sebelum** (baseline, langkah M0) dan **sesudah** (M5) dengan perintah yang sama:
- jumlah container aplikasi & total container;
- `docker stats --no-stream` (RAM & CPU per container, saat idle dan saat e2e berjalan);
- ukuran image (`docker images`);
- waktu build dari cache kosong dan dari cache hangat;
- waktu e2e penuh;
- jumlah panggilan HTTP internal yang dihapus;
- estimasi kebutuhan server minimum (RAM/CPU) dengan dan tanpa OCR.

---

# BAGIAN II — CR-012 AUTH DUA VERSI & PERSIAPAN SSO

> Dikerjakan di `main` setelah Bagian I (M5), sebelum fase F.

## 10. Tafsiran "2 versi auth"

| Versi | Nama | Isi | Dipakai oleh |
|---|---|---|---|
| **A** | **Auth Kampus (Core/SSO)** | Data dasar akun (`users`), katalog layanan/produk (`service_accounts`), hak akses akun ke layanan (`user_accounts`), kredensial, sesi, token, kunci & JWKS, lockout, reset password, admin user, role & permission per layanan, endpoint OpenID Connect (SSO) | Semua produk kampus (PMB sekarang; produk lain nanti, mis. **Pelita Sekolah** — BK digital & asesmen) |
| **B** | **Auth PMB** | Ekstensi khusus aplikasi pendaftaran di atas A: registrasi mandiri calon mahasiswa + OTP, ganti email saat pending, persetujuan ketentuan, pembersihan akun pending, perpindahan `calon_mahasiswa` → `mahasiswa`, dan seluruh endpoint `/api/v1/auth/*` yang dipakai frontend PMB sekarang | Aplikasi sistem pendaftaran saja |

**B bergantung pada A; A tidak boleh bergantung pada B.** A harus dapat di-build & dijalankan sendiri.

## 11. Skema data dari lead (versi A)

Lead menetapkan tiga tabel inti. Nama tabel di database mengikuti istilah lead:

| Istilah lead | Tabel | Isi |
|---|---|---|
| **UserTabel** | `users` | Data dasar akun: `id` (UUID global), `email` (unik), `password_hash`, `full_name`, `phone`, `status`, `email_verified_at`, `last_login_at`, `password_changed_at`, `terms_accepted_at`, timestamps. **Satu baris per orang**, berlaku di semua layanan |
| **ServiceAccount** | `service_accounts` | Katalog layanan/produk yang dapat diakses, mis. **Layanan Pendaftaran** (portal calon mahasiswa), **Layanan Admin Pendaftaran** (portal panitia), **Layanan Keuangan**, dan produk lain kelak. Kolom: `id`, `code` (unik, mis. `pmb-pendaftaran`, `pmb-admin`, `keuangan`), `name`, `description`, `product_group` (mis. `pmb`), `base_url`, `audience` (nilai `aud` token, mis. `pmb-api`), konfigurasi klien SSO: `client_type` (`public`+PKCE / `confidential`), `client_secret_hash` (nullable), `redirect_uris`, `post_logout_redirect_uris`, `allowed_scopes`, `first_party`, `is_active` |
| **UserAccount** | `user_accounts` | Penghubung: layanan apa saja yang boleh diakses seorang user. Kolom: `id`, `user_id`, `service_account_id`, `status` (`active`/`suspended`/`revoked`), `granted_by`, `granted_at`, `last_access_at`. Unik `(user_id, service_account_id)` |

Penyesuaian dengan RBAC yang sudah ada (CR-009):
- `roles` dan `permissions` mendapat kolom `service_account_id` (role milik layanan tertentu). Permission yang sudah ada **kodenya tidak diganti**.
- `user_roles` **[UBAH struktur, perilaku sama]** menjadi `user_account_roles` (`user_account_id`, `role_id`, `scope_type`, `scope_id`, `created_by`), sehingga role selalu melekat pada akses user ke layanan tertentu.
- **Migrasi data tanpa kehilangan data:** seeder membuat service account `pmb-pendaftaran` (role `calon_mahasiswa`, `mahasiswa`) dan `pmb-admin` (role `superadmin`, `admin_pmb`, `keuangan`, `prodi`, `dekan`, `pimpinan`, `dosen`), `pmb-mitra` (portal guru BK; role `guru_bk` ditambahkan di fase F), serta `keuangan` dengan `is_active=false` sebagai contoh produk berikutnya. Setiap baris `user_roles` lama dipindahkan menjadi `user_accounts` + `user_account_roles` pada layanan yang sesuai.
- Registrasi calon mahasiswa (versi B) membuat `users` + `user_accounts` (`pmb-pendaftaran`) + role `calon_mahasiswa`. Pembuatan staf oleh superadmin membuat `user_accounts` (`pmb-admin`) + role-nya.
- Akses ditolak bila user tidak punya `user_accounts` aktif untuk layanan yang diminta (`403 SERVICE_ACCESS_DENIED`). Menonaktifkan akses satu layanan tidak memengaruhi layanan lain; menonaktifkan `users.status` memutus semua layanan.

## 12. Struktur kode (di dalam service `gateway`)

```
services/gateway/
  cmd/gateway/main.go        # deployment PMB: proxy + auth core (A) + auth pmb (B)   <- yang berjalan
  cmd/sso/main.go            # deployment SSO mandiri: auth core (A) saja             <- disiapkan
  internal/proxy/...
  internal/auth/core/...     # VERSI A  (+ api.go sebagai facade)
  internal/auth/pmb/...      # VERSI B  (memakai facade core)
  migrations/core/  migrations/pmb/
```
- Arch test: `core` tidak mengimpor `pmb`/`proxy`; `cmd/sso` tidak mengimpor `pmb`/`proxy`.
- Database tetap `pmb_auth`, dibagi schema `auth_core` (users, service_accounts, user_accounts, roles, permissions, role_permissions, user_account_roles, sessions, password_resets, signing_keys, audit_logs) dan `auth_pmb` (user_otps, data khusus registrasi). Pemindahan tabel lewat migration tanpa kehilangan data.

## 13. Perilaku PMB tetap (versi B)
- Semua endpoint `/api/v1/auth/*` (register, verify, resend, change-email, login, refresh, logout, me, profile, password, forgot/reset, `me/permissions`, admin, internal) **tetap sama**.
- `POST /auth/login` tetap tanpa parameter wajib baru. Parameter **opsional** `service` (kode service account, mis. `pmb-admin`) boleh dikirim portal admin untuk memastikan akses; tanpa parameter → perilaku lama (gabungan role semua layanan `pmb`).
- Token PMB (`aud=pmb-api`, `roles`, `perms`, `scopes`, `sid`) **identik** dengan sebelumnya untuk user yang sama; klaim baru hanya tambahan: `svc` (daftar kode layanan yang dapat diakses), dan klaim standar OIDC di 14.3.
- Admin API baru (superadmin): CRUD `service_accounts` (termasuk rotasi secret klien), beri/cabut akses `user_accounts`, kelola role per layanan. Endpoint admin user yang lama tetap bekerja (di belakang layar memakai tabel baru).

## 14. Persiapan SSO (versi A)

### 14.1 Satu akun untuk banyak produk
`users.id` adalah identitas global. Calon mahasiswa yang menjadi mahasiswa **memakai akun yang sama**; kelak cukup ditambah `user_accounts` ke produk lain.

### 14.2 Service account sebagai klien SSO
Setiap `service_accounts` sekaligus klien OpenID Connect (kolom klien di Bagian 11). `pmb-pendaftaran` dan `pmb-admin`: `public` + PKCE, `first_party=true`.

### 14.3 Token kompatibel OpenID Connect
- `iss` terkonfigurasi (`AUTH_ISSUER`), `sub` = `users.id`, klaim standar `email_verified`, `name`, `preferred_username`; **ID token** dengan `nonce`, `auth_time`, `at_hash`.
- `/.well-known/openid-configuration`; JWKS tetap.
- **Rotasi kunci**: tabel `signing_keys` (metadata `kid` + public key; private key tetap dari file/secret), JWKS memuat kunci aktif + sebelumnya, `make rotate-keys`; token lama valid sampai kedaluwarsa.
- Token untuk `aud` tertentu hanya memuat role & permission layanan dengan audience tersebut.

### 14.4 Endpoint SSO — disiapkan, aktif bila `SSO_ENABLED=true`
| Endpoint | Keterangan |
|---|---|
| `GET /oauth2/authorize` | Hanya `code` + **PKCE S256 wajib**; `state`, `nonce`; `redirect_uri` harus persis terdaftar di service account; user wajib punya `user_accounts` aktif untuk layanan itu (bila tidak → halaman "Anda tidak memiliki akses ke layanan ini") |
| `POST /oauth2/token` | `authorization_code` (sekali pakai, ≤ 60 detik) dan `refresh_token` (rotasi + deteksi pemakaian ulang) |
| `GET /oauth2/userinfo` | Klaim sesuai scope `openid profile email` |
| `POST /oauth2/revoke` | Cabut refresh token |
| `GET/POST /oauth2/logout` | Akhiri sesi SSO + sesi produk turunannya, redirect ke URI terdaftar |
| `GET/POST /sso/login` | Halaman login terpusat **sementara** (HTML server, Bahasa Indonesia, logo & nama dari profil institusi), CSRF, logika login sama (lockout, anti-enumerasi, `remember_me`). Desain final dari Wahhed menyusul |

Sesi SSO: cookie `sso_session` (`HttpOnly`, `Secure`, `SameSite=Lax`), di Redis, idle default 8 jam, absolut 7 hari (30 hari bila `remember_me`).

### 14.5 Belum dikerjakan (rencana di `docs/SSO.md`)
Menjalankan `cmd/sso` sebagai service ke-4 di produksi (saat produk kedua dibuat — kandidat terdekat **Pelita Sekolah**, sehingga guru BK cukup satu akun untuk portal mitra PMB dan Pelita Sekolah; masih dalam batas 3–4 service dari lead), memindahkan login portal PMB ke alur redirect SSO, MFA, login Google/Microsoft, sinkronisasi LDAP/AD, layar persetujuan pihak ketiga, single logout back-channel.

## 15. Deployment Bagian II
- Compose development: `gateway` dengan `SSO_ENABLED=true` (supaya teruji) + profile `sso` untuk menjalankan `cmd/sso` mandiri (smoke test).
- Compose produksi: `SSO_ENABLED=false`, tanpa service tambahan. **Total service aplikasi tetap 3.**
- `.env.example`: `AUTH_ISSUER`, `SSO_ENABLED`, `SSO_SESSION_IDLE`, `SSO_SESSION_ABSOLUTE`, `SSO_SESSION_ABSOLUTE_REMEMBER`, `SSO_COOKIE_DOMAIN`.

## 16. Pengujian Bagian II
Wajib:
- e2e lama lulus **tanpa diubah**; jumlah PASS tidak berkurang.
- **Kompatibilitas token**: fixture klaim token PMB (diambil di M0) untuk beberapa role identik setelah perubahan, kecuali klaim tambahan (`svc` dan klaim standar).
- **Migrasi data**: semua `user_roles` lama muncul sebagai `user_accounts` + `user_account_roles` yang benar; jumlah baris & hak akses sebelum = sesudah (test migrasi dengan data contoh).
- Arch test versi A/B.

Skenario e2e yang **ditambahkan di akhir** `scripts/e2e.sh`:
1. Calon mahasiswa baru → punya akses `pmb-pendaftaran` saja; login dengan `service=pmb-admin` → `403 SERVICE_ACCESS_DENIED`.
2. Superadmin memberi akses `pmb-admin` + role `keuangan` ke user → user bisa masuk portal admin; cabut akses → ditolak, akses `pmb-pendaftaran` tetap.
3. Discovery & JWKS valid.
4. Alur SSO penuh dengan klien `pmb-pendaftaran`: authorize (PKCE) → `/sso/login` → kode → token → userinfo → ID token dengan `nonce` sama.
5. `code_verifier` salah, `redirect_uri` tak terdaftar, kode dipakai dua kali → ditolak.
6. Sesi SSO: authorize ke `pmb-admin` tanpa kata sandi lagi selama sesi aktif (bila user punya aksesnya); setelah logout SSO → diminta login.
7. Rotasi kunci: token lama tetap valid, token baru memakai `kid` baru.
8. `SSO_ENABLED=false` → `/oauth2/*` dan `/sso/*` 404, endpoint PMB normal.
9. Smoke test `cmd/sso` mandiri (profile `sso`).

## 17. Dokumentasi Bagian II
`docs/SSO.md` (konsep versi A & B, skema `users`–`service_accounts`–`user_accounts` dengan ERD Mermaid, alur OIDC, cara mendaftarkan produk baru, rencana 14.5, panduan validasi token untuk produk baru), pembaruan `ARCHITECTURE.md`, `RBAC.md` (kolom layanan), `class-diagram-gateway.md`, `erd-gateway.md`, `API.md` bagian "Layanan & Akses" dan "SSO".

---

# PELAKSANAAN GABUNGAN

## 7. Langkah kerja (milestone)

Semua langkah di branch `main`, beruntun mengikuti aturan Bagian 0.3; tidak ada titik tunggu persetujuan kecuali kondisi berhenti 0.7.

| Langkah | Isi | Selesai bila |
|---|---|---|
| **R0** Audit | Catat CR-011..CR-014 (open) & keputusan Bagian 8; buat `docs/BUGS.md`; audit mandiri 21.2 | BUGS.md terisi; commit "docs: CR-013 bug triage" |
| **R1** Perbaikan bug & jalur gratis | Semua P0/P1 diperbaiki; jalur gratis (26); `scripts/smoke-camaba.sh` | e2e FAIL: 0, smoke lulus |
| **R2** Duitku siap pakai | Bagian 22 seluruhnya | Test callback/rekonsiliasi lulus |
| **R3** Siap demo | Bagian 23–24; CR-013 done; tag `v0.5.1-rilis`; laporan fase R | `final-check -Tahap rilis` AMAN |
| **M0** Baseline & inventaris | Tag `pre-consolidation`; metrik 6.4 "sebelum"; e2e (catat X); inventaris `/internal/*`, job, database & tabel; **fixture klaim token PMB** per role; pemetaan auth A/B; rencana migrasi `user_roles`; pilih library OIDC (bandingkan minimal 2, catat alasan di DECISIONS.md) | Laporan di CONSOLIDATION-REPORT.md + fixture tersimpan |
| **M1** Kerangka pmb-service | `services/pmb-service` (platform, job runner, server, arch test), database `pmb_app`, compose menambah pmb-service berdampingan | Build & e2e tetap lulus |
| **M2** Pindah modul satu per satu | organization → identity → notification → admission → finance → exam → scoring → selection → student → nim; route gateway dialihkan, HTTP internal diganti facade, service lama dihapus dari compose. Facade `finance` untuk referal (35) boleh disiapkan di sini | **Satu commit per modul**, e2e FAIL: 0 di setiap commit |
| **M3** Gabung auth ke gateway | Kode auth pindah ke `services/gateway/internal/auth`, validasi JWT lokal, port internal, hapus auth-service | e2e FAIL: 0 |
| **M4** OCR opsional & pembersihan | `docker-compose.prod.yml` ditulis ulang untuk 3 service + profile `ocr`, `OCR_ENABLED`, hapus folder lama & `_template`, rapikan `go.work`, Makefile, `.env.example`, `docs/DEPLOY.md`, resource limit | Hanya 3 folder di `services/` |
| **M5** Dokumentasi & ukur | `ARCHITECTURE.md`, class diagram & ERD service baru, README, metrik "sesudah", CR-011 done, tag `v0.6.0-3-services` | e2e FAIL: 0 |
| **M6** Auth dua versi + skema lead | Pecah `auth/core` & `auth/pmb` + arch test; tabel `service_accounts`, `user_accounts`, `user_account_roles`; migrasi data `user_roles`; seeder layanan (termasuk `pmb-mitra`); admin API layanan & akses; parameter `service` opsional; `403 SERVICE_ACCESS_DENIED`; uji kompatibilitas token & migrasi | e2e lama FAIL: 0 + skenario 16 butir 1–2 |
| **M7** Persiapan SSO | Token OIDC-kompatibel, discovery, `signing_keys` + rotasi, `/oauth2/*` + `/sso/login` dengan library terpilih, sesi SSO, `SSO_ENABLED`, `cmd/sso` + profile `sso` | Skenario 16 butir 3–9 |
| **M8** Penutupan K | `docs/SSO.md` & dokumen terkait, CR-012 done, tag `v0.7.0-sso-ready`, laporan fase K | `final-check -Tahap konsolidasi` AMAN |
| **F1** Data, akun & akses | Modul `referral` di pmb-service (schema, migration, facade), role `guru_bk` di service account `pmb-mitra` + permission 34.1, route gateway, form akun guru BK + komisi, aktivasi, impor CSV, `referral_settings`, kolom `referral_commission_enabled` di master jalur | e2e 38 butir 1, 10, 12 |
| **F2** Kode & verifikasi | Cek & pasang kode (consent), ditagihkan ke mitra, verifikasi/tolak (satuan & massal), `SettleByPartner`, kunci kode, jalur gratis & bayar sendiri (33.2) | e2e 38 butir 2–4, 7, 11 |
| **F3** Komisi | Perhitungan & snapshot komisi, jalur tanpa komisi, mode penyelesaian umum + override per kode, pembayaran komisi | e2e 38 butir 5, 6, 9 |
| **F4** Setoran & portal | Ringkasan 33.5, daftar pengguna, ekspor, setoran & verifikasi, buku besar, notifikasi 36 | e2e 38 butir 8, 13 |
| **F5** Penutupan F | Dokumentasi 39; data demo guru BK; CR-014 done; tag `v0.8.0-referal`; laporan fase F | `final-check -Tahap penuh` AMAN |

## 8. Keputusan (catat di DECISIONS.md)

| Kode | Keputusan |
|---|---|
| R1 | Urutan: kesiapan rilis aplikasi pendaftaran (R) → pemangkasan 3 service & auth dua versi/SSO (K) → referal & komisi guru BK (F), semua di `main`; demo Selasa dari tag `v0.5.1-rilis` |
| R2 | Demo memakai Duitku sungguhan sesuai `DUITKU_ENV` di `.env` (sandbox atau produksi); kredensial disiapkan tim, tidak pernah masuk repo/log |
| R3 | Rilis hanya bila tidak ada bug P0/P1 Open di `docs/BUGS.md` |
| R4 | Freeze 19–21 Okt: hanya perbaikan P0/P1 |
| R5 | Jalur dengan biaya Rp 0 tidak melalui pembayaran; tagihan Rp 0 langsung lunas (metode `free`) agar alur berikutnya tetap konsisten |
| F1 | Siswa membayar ke guru BK di luar sistem; verifikasi guru BK melunasi tagihan formulir (`referral_collective`); guru BK menyetor ke kampus dan keuangan memverifikasi |
| F2 | Komisi per kode: persen per transaksi atau nominal per mahasiswa terverifikasi, disnapshot per transaksi; jalur gratis & jalur yang dinonaktifkan tidak berkomisi |
| F3 | Penyelesaian komisi (`terpisah` dibayar keuangan atau `potong_setoran`) diatur admin: pengaturan umum + override per kode |
| F4 | Akun guru BK hanya dibuat admin kampus (form atau impor CSV) dengan email aktivasi |
| F5 | NIK disamarkan untuk guru BK; dokumen & biodata lengkap tidak dapat diakses guru BK; siswa memberi persetujuan saat memasang kode |
| F6 | Kartu ringkasan mengikuti 33.5 ("Antrian Verifikasi Pembayaran" = pembayaran siswa yang menunggu verifikasi guru BK) |
| F7 | Referal dibangun langsung sebagai modul `referral` di `pmb-service`; role `guru_bk` milik service account `pmb-mitra` |
| C1 | 3 service: `gateway` (gateway + auth), `pmb-service` (11 modul bisnis termasuk `referral`), `ocr-service` |
| C2 | `pmb-service` modular monolith: schema per modul, facade antar modul, tanpa query/transaksi lintas schema |
| C3 | Auth digabung ke gateway agar identitas & pintu masuk dapat dipakai bersama produk kampus berikutnya |
| C4 | OCR opsional, dimatikan di luar periode pendaftaran |
| C5 | Tidak ada migrasi data antar database karena belum produksi (migrasi data auth di dalam `pmb_auth` tetap dilakukan) |
| C6 | API publik, kontrak JSON, dan perilaku tidak berubah; frontend tidak perlu diubah |
| S1 | Auth versi A (Kampus/SSO, umum) dan versi B (PMB, khusus pendaftaran) dalam satu codebase; B bergantung pada A |
| S2 | Skema akun mengikuti lead: `users` (UserTabel), `service_accounts` (ServiceAccount), `user_accounts` (UserAccount); role melekat pada akses user ke layanan |
| S3 | SSO memakai OpenID Connect (Authorization Code + PKCE); service account sekaligus klien SSO |
| S4 | SSO disiapkan & diuji sekarang, dimatikan di produksi; dijalankan sebagai service mandiri saat produk kedua dibuat (kandidat: Pelita Sekolah) |
| S5 | Calon mahasiswa yang menjadi mahasiswa memakai akun yang sama; guru BK memakai akun yang sama untuk portal guru BK dan kelak Pelita Sekolah |
| S6 | Halaman login SSO sementara dibuat sederhana di server sampai ada desain |

## 9. Definition of Done

**Fase R**
- [ ] Alur camaba Bagian 20 (termasuk jalur gratis) lulus e2e & smoke; `docs/BUGS.md` tanpa P0/P1 Open; pesan error berbahasa Indonesia.
- [ ] Duitku: `DUITKU_ENV`, pengaman konfigurasi, callback idempoten, rekonsiliasi, `docs/DUITKU-PRODUKSI.md`.
- [ ] `deploy/docker-compose.prod.yml`, `docs/DEPLOY.md` (termasuk demo dari tag), data demo, uji beban, `docs/RELEASE-CHECKLIST.md`, `docs/PANDUAN-ALUR.md`.
- [ ] CR-013 `done`, tag `v0.5.1-rilis`, `final-check -Tahap rilis` AMAN, laporan fase R.

**Fase K**
- [ ] `services/` hanya berisi `gateway`, `pmb-service`, `ocr-service`; compose development semua healthy; compose produksi dengan OCR opsional dan `SSO_ENABLED=false`.
- [ ] e2e lama (termasuk skenario R) lulus dengan jumlah PASS tidak berkurang (selain penggantian 6.3); kontrak JSON tetap valid; token PMB kompatibel (fixture).
- [ ] Arch test batas modul pmb-service dan batas auth versi A/B lulus; job runner aman multi-replika.
- [ ] Tabel `service_accounts`, `user_accounts`, `user_account_roles` berjalan; data akses lama termigrasi utuh; admin API layanan & akses tersedia.
- [ ] Endpoint SSO + halaman login sementara teruji; `cmd/sso` dapat dijalankan mandiri.
- [ ] Dokumen konsolidasi & SSO lengkap; CR-011 dan CR-012 `done`; tag `v0.6.0-3-services` & `v0.7.0-sso-ready`; `final-check -Tahap konsolidasi` AMAN; laporan fase K.

**Fase F**
- [ ] Modul `referral` di pmb-service: kode referal, verifikasi guru BK, komisi (dua tipe, mode penyelesaian umum + per kode, jalur tanpa komisi), setoran, akun guru BK oleh admin + impor CSV, notifikasi, RBAC `guru_bk`; e2e 38 lulus; arch test batas modul tetap lulus.
- [ ] `docs/REFERRAL.md` dan dokumen terkait; CR-014 `done`, tag `v0.8.0-referal`, `final-check -Tahap penuh` AMAN, laporan fase F.
