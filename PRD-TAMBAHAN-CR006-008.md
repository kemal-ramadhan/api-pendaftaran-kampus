# PRD Tambahan — CR-006, CR-007, CR-008

| | |
|---|---|
| **Produk** | Sistem PMB (backend) — repo `diaricakarsa/pmb-backend` |
| **Isi** | CR-006 Riwayat & Pindah Jalur · CR-007 Pengunduran Diri · CR-008 Uji Duitku Sandbox |
| **Sumber** | Arahan lead (pak Kemal), 29 Sep 2026: "History pindah jalur, opsi pengunduran diri" dan "Duitku bisa pakai yang sandbox, jadi gak usah verifikasi dokumen" |
| **Versi** | v1 — 29 September 2026 |
| **Hubungan dengan PRD utama** | Dokumen **terpisah**. `PRD-LENGKAP.md` (v3.3) tetap menjadi dasar semua aturan umum (format respons, keamanan, pola service, definisi selesai, prosedur push). Dokumen ini **tidak mengubah** isi `PRD-LENGKAP.md`; bila bertentangan, **dokumen ini yang berlaku** untuk tiga CR di bawah. |
| **Prasyarat** | Tahap 1–11 + CR-001..CR-005 sudah terverifikasi (final-check AMAN) |

---

## 0. Instruksi untuk AI Agent

1. Baca dokumen ini sampai habis, lalu baca bagian terkait di `PRD-LENGKAP.md`: C.4.4 (Admission), C.4.5 (Finance), C.4.6–C.4.9 (Exam, Scoring, Selection, Student, NIM), D.1–D.4 (Duitku), D.5–D.8 (CR-002 batal & daftar lagi), D.14–D.18 (CR-005 view endpoint).
2. Jangan mengubah perilaku yang sudah lulus e2e kecuali disebut di dokumen ini. Semua e2e lama wajib tetap lulus.
3. Ikuti `docs/AGENT_RULES.md` (aturan kerja, definisi selesai, prosedur commit & push, format laporan).
4. Semua keputusan default di Bagian 5 dicatat di `docs/DECISIONS.md` dengan tanda "default, menunggu konfirmasi lead".
5. Ketiga CR dicatat di `docs/CHANGE_REQUESTS.md` (status `open` → `done` setelah selesai dan terverifikasi).

---

## 1. CR-006 — Riwayat & Pindah Jalur

### 1.1 Tujuan
- Calon mahasiswa dapat **pindah jalur** (dan/atau prodi pilihan) dengan aturan yang jelas sesuai tahap pendaftarannya.
- Setiap perubahan tercatat sebagai **riwayat** yang tidak bisa diubah/dihapus, dapat dilihat calon mahasiswa dan admin.
- Riwayat juga mencakup pembatalan (CR-002), pendaftaran ulang setelah batal/tidak lulus (CR-002), dan pengunduran diri (CR-007), sehingga satu linimasa menggambarkan seluruh perjalanan pendaftar.

### 1.2 Aturan pindah jalur per tahap

| Status pendaftaran | Oleh calon mahasiswa | Oleh admin (`admin_pmb`) |
|---|---|---|
| `draft`, `awaiting_payment` | **Langsung** lewat endpoint pilihan yang sudah ada; tercatat di riwayat | Boleh, alasan wajib |
| `form_open`, `submitted`, `revision_required`, `verified` (sudah bayar, belum dijadwalkan ujian) | **Ajukan permintaan** → disetujui/ditolak `admin_pmb` | Boleh langsung, alasan wajib |
| `exam_scheduled` (belum ada nilai) | Tidak boleh (`PATH_CHANGE_NOT_ALLOWED`) | Boleh, alasan wajib; peserta dikeluarkan dari sesi ujian lama |
| Sudah ada nilai, `passed`, `failed`, `reregistered`, `enrolled`, `cancelled`, `withdrawn` | Tidak boleh | Tidak boleh (`PATH_CHANGE_NOT_ALLOWED`) |

Aturan tambahan:
1. "Pindah jalur" mencakup perubahan `path_id`, `first_choice_program`, `second_choice_program`, dan `period_id`. Kombinasi baru harus dibuka di gelombang tersebut dan periode pendaftaran masih buka (`PERIOD_CLOSED`).
2. **Batas pindah setelah bayar:** maksimal 1 kali disetujui per pendaftaran (`PATH_CHANGE_MAX_AFTER_PAYMENT=1`). Melebihi → `PATH_CHANGE_LIMIT_REACHED`. Pindah saat `draft`/`awaiting_payment` tidak dibatasi.
3. Hanya boleh ada **satu permintaan `pending`** per pendaftaran (`PATH_CHANGE_PENDING`). Calon mahasiswa boleh membatalkan permintaannya sendiri selama masih `pending`.
4. Saat `awaiting_payment` dan ada pembayaran Duitku `pending` yang belum kedaluwarsa → tolak `409 PAYMENT_IN_PROGRESS` beserta `expired_at` pembayaran itu. Bila biaya pendaftaran jalur baru berbeda, tagihan lama dibatalkan dan tagihan baru dibuat (lewat `POST /internal/invoices/:id/reprice` di Finance, idempoten, hanya bila belum ada pembayaran sukses/pending).
5. **Selisih biaya setelah bayar** (saat persetujuan):
   - Biaya jalur baru **lebih mahal** → Finance membuat tagihan selisih `INV-REG-{YYYY}-{urut}-ADJ`. Pengajuan verifikasi berikutnya wajib melunasi selisih (`PAYMENT_REQUIRED`).
   - Biaya jalur baru **lebih murah** → tidak ada pengembalian di sistem; selisih dicatat di riwayat dan keuangan diberi notifikasi (sesuai keputusan "tidak ada refund di sistem").
6. **Dokumen** setelah persetujuan: persyaratan dihitung ulang untuk jalur baru. Dokumen dengan kode yang masih dipersyaratkan tetap dipakai beserta statusnya. Bila ada persyaratan baru yang belum diunggah → status kembali `form_open`; bila lengkap → status `submitted` (diverifikasi ulang admin).
7. Pindah saat `exam_scheduled` oleh admin: Exam Service mengeluarkan peserta dari sesi lama (internal API) dan admin menjadwalkan ulang. Nomor pendaftaran **tidak berubah** pada pindah jalur.
8. Semua perubahan lintas service memakai outbox (idempoten), sesuai `PRD-LENGKAP.md` C.5.

### 1.3 Data (Admission Service, migration baru)

- `application_changes` — riwayat, **hanya insert** (repository tidak menyediakan update/delete; tambahkan trigger DB yang menolak `UPDATE`/`DELETE`):
  `id`, `application_id`, `user_id`, `change_type` (`created`, `choices_updated`, `path_change_requested`, `path_change_approved`, `path_change_rejected`, `path_change_cancelled`, `admin_moved`, `cancelled`, `reapplied`, `withdrawal_requested`, `withdrawal_approved`, `withdrawal_rejected`, `withdrawal_cancelled`), `from_values` (jsonb: period/path/program beserta label), `to_values` (jsonb), `reason`, `fee_difference` (integer, bisa negatif), `actor_id`, `actor_role`, `source` (`applicant`|`admin`|`system`), `related_request_id`, `created_at`.
- `path_change_requests`: `id`, `application_id`, `user_id`, `from_*` dan `to_*` (period, path, first choice, second choice), `reason`, `status` (`pending`|`approved`|`rejected`|`cancelled`), `fee_difference`, `reviewed_by`, `review_note`, `reviewed_at`, `created_at`.
- `applications` tambah: `path_change_count_after_payment` (int), `previous_application_id` (nullable; diisi saat pendaftaran baru dibuat setelah batal/tidak lulus/mundur).
- **Backfill:** buat entri `created`, `cancelled`, `reapplied` untuk data yang sudah ada dari `application_status_logs` (migration data), supaya riwayat lama ikut tampil.

### 1.4 Endpoint

**Calon mahasiswa** (lewat gateway `/api/v1/admission`)
| Method | Path | Keterangan |
|---|---|---|
| GET | `/applications/me/changes` | Linimasa riwayat seluruh pendaftaran milik user (terbaru dulu, paginasi). Tiap item: `change_type`, `title_label` (mis. "Pindah jalur dari Reguler ke Prestasi"), `description`, `actor_label` ("Anda" / "Admin PMB" / "Sistem"), `reason`, `fee_difference`, `application_registration_number`, `created_at`, `created_at_label` |
| GET | `/applications/me/path-change/options` | Pilihan jalur/prodi yang boleh dituju + perkiraan `fee_difference` + `allowed` dan `block_reason_label` |
| POST | `/applications/me/path-change` | Pada `draft`/`awaiting_payment` → langsung diterapkan. Pada status setelah bayar → membuat permintaan `pending`. Body: `period_id?`, `path_id`, `first_choice_program`, `second_choice_program?`, `reason` (wajib bila setelah bayar, 10–500 karakter) |
| DELETE | `/applications/me/path-change/pending` | Batalkan permintaan `pending` milik sendiri |

**Admin** (`admin_pmb`; baca juga `superadmin`, `pimpinan`; `prodi` hanya baca untuk prodinya)
| Method | Path | Keterangan |
|---|---|---|
| GET | `/admin/path-change-requests?status=pending` | Daftar permintaan + filter + paginasi |
| POST | `/admin/path-change-requests/:id/approve` | Body `note?` |
| POST | `/admin/path-change-requests/:id/reject` | Body `note` (wajib) |
| POST | `/admin/applications/:id/move` | Pindah oleh admin, body `path_id`, `first_choice_program`, `second_choice_program?`, `period_id?`, `reason` (wajib) |
| GET | `/admin/applications/:id/changes` | Riwayat lengkap satu pendaftar (termasuk pendaftaran sebelumnya) |
| GET | `/admin/path-changes/export` | xlsx |

**Internal baru:** Finance `POST /internal/invoices/:id/reprice`, `POST /internal/invoices/adjustment`; Exam `POST /internal/participants/remove`.

**Kompatibilitas CR-005:** `GET /applications/me/dashboard` menambah field `pending_request` (`{type: "path_change"|"withdrawal", status_label, created_at}` atau `null`). Field lama tidak berubah.

Notifikasi baru: `path_change_requested` (ke admin), `path_change_approved`, `path_change_rejected`, `admin_moved` (ke calon mahasiswa).

Error code baru: `PATH_CHANGE_NOT_ALLOWED`, `PATH_CHANGE_LIMIT_REACHED`, `PATH_CHANGE_PENDING`, `PAYMENT_IN_PROGRESS`.

---

## 2. CR-007 — Pengunduran Diri

### 2.1 Tujuan
Calon mahasiswa yang **sudah membayar** biaya pendaftaran dapat mengundurkan diri secara resmi, tercatat, dan berdampak benar ke tagihan, jadwal ujian, kuota, NIM, dan role. (Sebelum bayar tetap memakai **Batalkan pendaftaran** dari CR-002.)

### 2.2 Aturan per tahap

| Status | Pengunduran diri |
|---|---|
| `draft`, `awaiting_payment` | Tidak tersedia → gunakan pembatalan CR-002 (`WITHDRAWAL_NOT_ALLOWED`, pesan mengarahkan ke Batalkan) |
| `form_open`, `submitted`, `revision_required`, `verified`, `exam_scheduled` | **Langsung berlaku** setelah konfirmasi |
| `passed`, `reregistered`, `enrolled` (NIM terbit, mahasiswa belum aktif) | **Perlu persetujuan** `admin_pmb` |
| Mahasiswa sudah `active` | Tidak tersedia di PMB (urusan akademik) → `WITHDRAWAL_NOT_ALLOWED` dengan pesan menghubungi bagian akademik |
| `cancelled`, `failed`, `withdrawn` | Tidak tersedia |

Aturan tambahan:
1. **Konfirmasi wajib:** kategori alasan, teks alasan (10–500 karakter), centang pernyataan, dan **kata sandi akun** (diverifikasi lewat Auth internal `POST /internal/users/:id/verify-password`; gagal dihitung ke lockout login yang sama).
2. Kategori alasan: `accepted_elsewhere` (diterima di kampus lain), `financial` (biaya), `family` (keluarga), `health` (kesehatan), `relocation` (pindah domisili), `work` (bekerja), `other` (lainnya). Label Bahasa Indonesia dikirim backend.
3. Selama permintaan `pending`: aksi lanjutan (bayar, daftar ulang, unggah dokumen) ditolak `409 WITHDRAWAL_PENDING`. Calon mahasiswa boleh membatalkan permintaannya sendiri.
4. **Saat berlaku** (langsung atau disetujui), dalam satu alur idempoten (outbox):
   - Status pendaftaran → `withdrawn` (status akhir baru; tidak aktif).
   - Finance: tagihan/termin **belum dibayar** dibatalkan; pembayaran Duitku `pending` dibatalkan (callback sukses yang datang kemudian → `paid_after_cancel`, sama seperti CR-002). Pembayaran yang **sudah masuk** tidak dikembalikan otomatis; ditandai `refund_review` dan muncul di daftar keuangan untuk ditangani manual.
   - Exam: peserta dikeluarkan dari sesi yang belum berlangsung; attempt CBT yang sedang berjalan dikumpulkan otomatis.
   - Selection: hasil ditandai `withdrawn`; kuota prodi yang terpakai dilepas; `prodi` dan `admin_pmb` diberi notifikasi bahwa ada kursi kosong (promosi dari `waitlist` tetap lewat alur Selection yang ada, tidak otomatis).
   - Student: daftar ulang dibatalkan.
   - NIM: bila sudah terbit → status `voided` (tidak dipakai ulang; urutan tidak dimundurkan).
   - Auth: role `mahasiswa` dicabut bila sempat diberikan (internal baru `DELETE /internal/users/:id/roles/:code`, idempoten).
5. **Bukti pengunduran diri** (PDF) tersedia setelah berlaku: nomor `{PREFIX}/UND/{YYYY}/{urut}`, nama, nomor pendaftaran, prodi/jalur, tahap saat mundur, tanggal berlaku, kategori alasan, QR verifikasi (`GET /admission/withdrawals/verify/:code`, publik, rate limited, data termask).
6. **Setelah mundur:** boleh membuat pendaftaran baru bila ada gelombang yang masih buka (aturan eligibilitas CR-002; `withdrawn` diperlakukan seperti `cancelled`). Pendaftaran baru terhubung lewat `previous_application_id` dan muncul di riwayat CR-006.
7. **Pengunduran diri oleh admin** (mis. surat tertulis diterima langsung): admin membuat pengunduran diri atas nama pendaftar dengan alasan dan unggahan surat (PDF/JPG ≤ 5 MB) → langsung berlaku; tercatat dengan `source = admin`.

### 2.3 Data (Admission, migration baru)
- `withdrawals`: `id`, `application_id`, `user_id`, `stage_at_request`, `reason_category`, `reason_text`, `status` (`pending`|`approved`|`rejected`|`cancelled`|`effective`), `requires_approval`, `requested_at`, `reviewed_by`, `review_note`, `reviewed_at`, `effective_at`, `source` (`applicant`|`admin`), `attachment_object_key`, `letter_number`, `letter_object_key`, `verify_code_hash`.
- Status pendaftaran baru: `withdrawn`. Perbarui state machine dan dokumentasinya.
- Finance: status pembayaran tambah `refund_review`; tabel/daftar tinjauan pengembalian untuk `keuangan` (`GET /finance/admin/refund-reviews`, `PATCH .../:id` dengan catatan penyelesaian manual).
- NIM: status `voided` pada `nim_issuances`.

### 2.4 Endpoint

**Calon mahasiswa** (`/api/v1/admission`)
| Method | Path | Keterangan |
|---|---|---|
| GET | `/applications/me/withdrawal/eligibility` | `{allowed, requires_approval, block_reason_label, consequences: [..daftar dampak dalam Bahasa Indonesia..], reason_categories: [{code,label}]}` untuk ditampilkan sebelum konfirmasi |
| POST | `/applications/me/withdrawal` | Body `reason_category`, `reason_text`, `agree_statement: true`, `password` |
| GET | `/applications/me/withdrawal` | Status permintaan terakhir |
| DELETE | `/applications/me/withdrawal/pending` | Batalkan permintaan `pending` |
| GET | `/applications/me/withdrawal/letter` | PDF bukti pengunduran diri (setelah berlaku) |

**Admin**
| Method | Path | Akses |
|---|---|---|
| GET | `/admin/withdrawals?status=` | `admin_pmb`, `superadmin`, `pimpinan` (baca), `prodi` (baca, prodinya) |
| POST | `/admin/withdrawals/:id/approve` \| `/reject` | `admin_pmb` (reject: catatan wajib) |
| POST | `/admin/applications/:id/withdraw` | `admin_pmb` (atas nama pendaftar, multipart dengan surat) |
| GET | `/admin/withdrawals/stats` | Jumlah per kategori alasan, per tahap, per prodi, per gelombang — untuk dashboard pimpinan |
| GET | `/admin/withdrawals/export` | xlsx |

Notifikasi baru: `withdrawal_requested` (ke admin), `withdrawal_effective`, `withdrawal_rejected`, `seat_released` (ke prodi & admin), `refund_review_created` (ke keuangan).

Error code baru: `WITHDRAWAL_NOT_ALLOWED`, `WITHDRAWAL_PENDING`, `PASSWORD_CONFIRMATION_FAILED`.

---

## 3. CR-008 — Uji Integrasi Duitku Sandbox Sungguhan

### 3.1 Latar
Lead: akun **sandbox** Duitku bisa dipakai tanpa verifikasi dokumen. Maka integrasi Duitku (CR-001, `PRD-LENGKAP.md` D.1–D.4, dan metode pembayaran CR-005 D.15.6) **wajib diuji terhadap sandbox Duitku sungguhan**, bukan hanya MockProvider.

### 3.2 Yang disiapkan manusia (Diar)
1. Merchant Code & API Key **sandbox**: minta akun sandbox tim ke pak Kemal, atau daftar di dashboard sandbox Duitku lalu buat project.
2. Disimpan **hanya** di `.env` lokal (tidak pernah di-commit, tidak dicetak di log/laporan).
3. URL publik untuk callback memakai tunnel, contoh (tanpa akun): `cloudflared tunnel --url http://localhost:8000` → isi `DUITKU_CALLBACK_URL=https://<acak>.trycloudflare.com/api/v1/finance/webhooks/payments/duitku`.

### 3.3 Yang dikerjakan agent
1. **Periksa ulang semua endpoint dan rumus signature di dokumentasi resmi Duitku** (docs.duitku.com) sebelum mengubah kode, lalu catat hasilnya (URL + rumus + tanggal dicek) di `docs/DUITKU-SANDBOX.md`. Ringkasan hasil pengecekan awal oleh tim (wajib diverifikasi ulang, dokumentasi resmi yang berlaku):
   | Fungsi | Sandbox URL | Signature (menurut dokumentasi saat dicek) |
   |---|---|---|
   | Create invoice (POP) | `https://api-sandbox.duitku.com/api/merchant/createInvoice` | Header `x-duitku-signature = HMAC_SHA256(merchantCode + timestamp, apiKey)` |
   | Daftar metode pembayaran | `https://sandbox.duitku.com/webapi/api/merchant/paymentmethod/getpaymentmethod` | `SHA256(merchantcode + amount + datetime + apiKey)` |
   | Cek status transaksi | `https://sandbox.duitku.com/webapi/api/merchant/transactionStatus` | `MD5(merchantCode + merchantOrderId + apiKey)` |
   | Callback | ke `DUITKU_CALLBACK_URL` | `HMAC_SHA256(merchantCode + amount + merchantOrderId, apiKey)` |
   Karena ada **dua host** (POP dan WebAPI), jadikan keduanya konfigurasi terpisah: `DUITKU_POP_BASE_URL`, `DUITKU_WEBAPI_BASE_URL` (default otomatis dari `DUITKU_ENV`). Bila dokumentasi resmi sudah memakai rumus yang lebih baru (mis. HMAC untuk semua endpoint), ikuti dokumentasi dan perbarui tabel.
2. Pemetaan error Duitku → `502 PAYMENT_PROVIDER_ERROR` (pesan Bahasa Indonesia; `statusCode`/`statusMessage` asli hanya di log tanpa API key).
3. Job rekonsiliasi dapat dipercepat untuk pengujian: `DUITKU_RECONCILE_INTERVAL` (default `10m`, uji `1m`).
4. Callback: verifikasi signature (constant-time), `merchantCode`, dan kecocokan `amount`; idempoten; payload mentah di `payment_webhook_logs`. Tambahkan opsi allowlist IP callback (`DUITKU_CALLBACK_IP_ALLOWLIST`, default kosong = nonaktif) bila Duitku mempublikasikan daftar IP.
5. **Script uji sandbox untuk Windows:** `scripts/duitku-sandbox-check.ps1` (PowerShell, ASCII saja, tanpa karakter khusus). Alur:
   1. Cek env Duitku terisi dan `PAYMENT_PROVIDER=duitku`, cek gateway sehat, cek `DUITKU_CALLBACK_URL` dapat diakses dari internet (GET health lewat URL tunnel).
   2. Buat calon mahasiswa uji lewat gateway (register → OTP dari Mailpit → verify), buat pendaftaran.
   3. `GET /finance/payment-methods` → pastikan daftar dari Duitku sandbox (bukan mock).
   4. Bayar tagihan pendaftaran dengan metode VA → tampilkan **nomor VA**, **nama bank**, dan `payment_url`.
   5. Minta penguji membayar di halaman simulasi pembayaran sandbox Duitku (URL demo VA sandbox dari dokumentasi), lalu **polling** status tiap 10 detik sampai 10 menit.
   6. PASS bila tagihan menjadi `paid` dan pendaftaran `form_open`; laporkan apakah lunas **lewat callback** atau **lewat rekonsiliasi**.
   7. Uji kedua (opsional flag `-TestReconcile`): bayar tanpa tunnel aktif → pastikan lunas lewat rekonsiliasi.
   8. Tulis `duitku-sandbox-report.txt` (API key dan merchant code dimasking) — file ini di `.gitignore`.
6. `scripts/e2e.sh` tetap memakai `PAYMENT_PROVIDER=mock` (otomatis, tanpa internet). Uji sandbox bersifat manual/opt-in.
7. `docs/DUITKU-SANDBOX.md`: langkah lengkap untuk manusia (dapatkan kredensial sandbox, isi `.env`, jalankan tunnel, jalankan script, cara membayar di simulator, cara membaca laporan, kembali ke mode mock).

---

## 4. Pengujian

### 4.1 Unit/integration test
- CR-006: matriks aturan per status (langsung / permintaan / ditolak), batas 1 kali setelah bayar, satu `pending`, `PAYMENT_IN_PROGRESS`, perhitungan selisih biaya (lebih mahal → tagihan ADJ; lebih murah → catatan + notifikasi), penghitungan ulang dokumen, riwayat hanya-insert (UPDATE/DELETE ditolak DB), backfill.
- CR-007: matriks per status, verifikasi kata sandi (salah → `PASSWORD_CONFIRMATION_FAILED` dan dihitung lockout), dampak lintas service idempoten (jalankan dua kali → hasil sama), NIM `voided` tidak dipakai ulang, role `mahasiswa` dicabut, `refund_review` terbentuk untuk pembayaran yang sudah masuk, balapan (race) antara pengunduran diri dan callback pembayaran.
- CR-008: signature ketiga endpoint + callback dengan vektor uji tetap; parsing respons sandbox (fixture dari respons nyata yang sudah dimasking); pemetaan error; pemilihan base URL per `DUITKU_ENV`.

### 4.2 e2e (lewat gateway, dari DB kosong, `PAYMENT_PROVIDER=mock`)
1. Pindah jalur saat `awaiting_payment` → langsung; tagihan diperbarui; riwayat memuat entri.
2. Bayar → ajukan pindah jalur → `pending`; ajukan lagi → `PATH_CHANGE_PENDING`; admin setujui → jalur baru; jalur lebih mahal → tagihan ADJ; submit sebelum ADJ lunas → `PAYMENT_REQUIRED`.
3. Ajukan pindah kedua setelah bayar → `PATH_CHANGE_LIMIT_REACHED`.
4. Status `exam_scheduled` → pindah oleh calon mahasiswa ditolak; admin memindah → peserta keluar dari sesi lama.
5. Riwayat: calon mahasiswa hanya melihat miliknya; linimasa memuat pembatalan, pendaftaran ulang, pindah jalur, dan pengunduran diri dengan label yang benar.
6. Mundur saat `verified` → langsung `withdrawn`; bukti PDF tersedia; verifikasi QR valid; tagihan belum bayar batal.
7. Mundur saat `passed` → `pending`; bayar selama `pending` → `WITHDRAWAL_PENDING`; admin setujui → `withdrawn`, pembayaran yang sudah masuk → `refund_review`, notifikasi `seat_released` ke prodi.
8. Mundur setelah NIM terbit (belum aktif) → disetujui → NIM `voided`, login ulang tanpa role `mahasiswa`.
9. Mundur saat `draft` → `WITHDRAWAL_NOT_ALLOWED`; mahasiswa `active` → `WITHDRAWAL_NOT_ALLOWED`; kata sandi salah → `PASSWORD_CONFIRMATION_FAILED`.
10. Setelah `withdrawn` → buat pendaftaran baru di gelombang yang masih buka → berhasil, terhubung `previous_application_id`.
11. Semua skenario lama tetap lulus. Ringkasan akhir tetap `Total PASS: X, FAIL: Y`.

### 4.3 Uji sandbox (manual, oleh Diar)
`scripts/duitku-sandbox-check.ps1` → PASS (lunas lewat callback), dan dengan `-TestReconcile` → PASS (lunas lewat rekonsiliasi).

---

## 5. Keputusan default (perlu konfirmasi lead)

| Kode | Keputusan |
|---|---|
| P1 | Pindah jalur oleh calon mahasiswa setelah bayar hanya lewat permintaan yang disetujui admin, maksimal 1 kali, dan hanya sebelum dijadwalkan ujian |
| P2 | Selisih biaya lebih mahal ditagihkan; lebih murah tidak dikembalikan otomatis (dicatat untuk keuangan) |
| P3 | Setelah pindah jalur, dokumen diverifikasi ulang bila jalur baru punya persyaratan berbeda |
| P4 | Nomor pendaftaran tetap saat pindah jalur |
| M1 | Pengunduran diri sebelum lulus langsung berlaku; setelah lulus/daftar ulang/NIM perlu persetujuan admin |
| M2 | Pengunduran diri wajib konfirmasi kata sandi dan alasan berkategori |
| M3 | Uang yang sudah dibayar tidak dikembalikan otomatis; ditandai untuk ditinjau keuangan |
| M4 | NIM yang sudah terbit dibatalkan (`voided`), tidak dipakai ulang |
| M5 | Setelah mundur, boleh mendaftar lagi di gelombang yang masih buka |
| M6 | Kursi yang kosong karena mundur dilaporkan ke prodi; promosi cadangan (waitlist) tetap manual |
| S1 | Uji otomatis tetap memakai mock; uji Duitku sandbox manual lewat script |

---

## 6. Catatan untuk frontend (koordinasi dengan Wahhed, bukan scope backend)
- Halaman/tab **Riwayat Pendaftaran** memakai `GET /applications/me/changes`.
- Tombol **Pindah Jalur** di halaman Jalur Pendaftaran memakai `path-change/options` (menampilkan perkiraan selisih biaya dan alasan bila tidak bisa).
- Tombol **Mengundurkan Diri** (mis. di Profil atau Daftar Ulang) menampilkan `withdrawal/eligibility.consequences` sebelum konfirmasi, lalu isian alasan + kata sandi.
- Banner status permintaan `pending` di Beranda memakai `dashboard.pending_request`.
- Belum ada desain Figma untuk tiga hal di atas; perlu dibuat Wahhed.

## 7. Definition of Done
- [ ] Semua endpoint Bagian 1.4 dan 2.4 jadi, terdokumentasi (swagger, `docs/API.md`, Postman, `docs/contracts/*.schema.json` untuk endpoint tampilan baru), dan teruji.
- [ ] Riwayat tidak bisa diubah/dihapus (dibuktikan test).
- [ ] Semua skenario 4.2 masuk `scripts/e2e.sh` dan lulus bersama skenario lama.
- [ ] `docs/DUITKU-SANDBOX.md` dan `scripts/duitku-sandbox-check.ps1` tersedia; unit test Duitku lulus.
- [ ] CR-006, CR-007, CR-008 berstatus `done` di `docs/CHANGE_REQUESTS.md`; keputusan P1–S1 di `docs/DECISIONS.md`.
- [ ] Class diagram & ERD Admission, Finance, Student, NIM diperbarui.
- [ ] `scripts/final-check.ps1` → AMAN.
