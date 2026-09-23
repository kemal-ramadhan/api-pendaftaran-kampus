# Prompt AI Agent — Fase 2 Bertahap Sesuai Timeline

**Cara pakai:** kirim prompt **satu tahap saja**. Tunggu agent melapor, lakukan cek manual (checklist di bawah setiap tahap), baru kirim tahap berikutnya. Setelah setiap tahap selesai, update baris yang sesuai di timeline Google Sheets.

Letakkan `PRD-fase-2.md` di root repo sebelum Tahap 0.

---

## Tahap 0 — Kickoff & aturan kerja (kirim sekali di awal)

```
Fase 1 (Auth + Gateway) selesai dan lulus e2e 43/43. Kita masuk fase 2
dengan acuan PRD-fase-2.md di root repo. Pekerjaan dilakukan BERTAHAP
mengikuti tabel "9.1 Pemetaan ke Timeline Tim". Aku akan mengirim
instruksi per tahap; jangan mengerjakan tahap yang belum aku minta.

Langkah sekarang (tanpa menulis kode service):
1. Baca PRD-fase-2.md (mulai Bagian 0), PRD.md, docs/DECISIONS.md,
   docs/API.md, docs/class-diagram-auth-service.md, serta kode di
   services/, pkg/, deploy/.
2. Buat docs/AGENT_RULES.md berisi aturan kerja berikut, lalu patuhi
   di setiap tahap:
   a. Jangan mengubah perilaku Auth dan Gateway yang sudah lulus e2e,
      kecuali yang disebut PRD-fase-2 Bagian 4.10 atau perbaikan bug.
   b. Setiap service baru: pola sama dengan auth-service, wajib
      pkg/gatewayauth, otorisasi role & scope di service, migration SQL,
      swagger, Dockerfile dengan BuildKit cache, .env.example,
      healthcheck, route di routes.yaml gateway.
   c. Sebelum melapor: go build, go vet, go test; docker compose down -v
      lalu up -d --build (semua healthy); scripts/e2e.sh semua PASS
      (skenario lama + skenario baru).
   d. Dokumentasi per tahap: docs/API.md, Postman collection,
      docs/class-diagram-<service>.md, docs/erd-<service>.md,
      DECISIONS.md, dan update status di PRD-fase-2 Bagian 3.
   e. Commit per milestone dengan pesan jelas, lalu push.
   f. Laporan: hasil test & e2e (PASS/FAIL), endpoint baru, keputusan
      baru, perintah verifikasi manual untukku. Lalu BERHENTI.
   g. Perubahan dari lead dicatat di docs/CHANGE_REQUESTS.md (nomor CR,
      tanggal, isi, tahap yang terdampak, status).
3. Buat docs/CHANGE_REQUESTS.md (kosong, dengan format tabel).
4. Laporkan: ringkasan pemahaman arsitektur fase 2, hal di PRD yang
   bertentangan dengan kode fase 1 (sebutkan lokasi), dan hal ambigu
   beserta usulan keputusan.
Commit "docs: agent rules & change request log", lalu BERHENTI.
```

---

## Tahap 1 — Organization Service (22–25 Sep)

```
Kerjakan Tahap 1 sesuai PRD-fase-2.md tabel 9.1: F2-M0 (Fondasi
bersama) lalu F2-M1 (Organization Service, Bagian 4.1 + validasi scope
prodi di Auth, Bagian 4.10). Ikuti docs/AGENT_RULES.md.

F2-M0: pkg/internalclient, pkg/storage (MinIO), pkg/pagination,
pkg/outbox, MinIO + init script database per service di docker-compose.
F2-M1: tabel faculties & study_programs, endpoint publik & admin,
endpoint internal, seeder 2 fakultas + 4 prodi, route gateway.

Tambahkan ke e2e.sh: daftar prodi publik; superadmin membuat prodi;
calon mahasiswa ditolak membuat prodi (403); superadmin memberi role
prodi dengan scope yang tidak ada → ditolak; dengan scope valid →
berhasil, dan token user tersebut memuat scope-nya.
Commit per milestone. Laporkan lalu BERHENTI.
```

**Cek manual:**
- [ ] e2e FAIL: 0, semua container healthy (termasuk MinIO dan organization-service)
- [ ] `GET localhost:8000/api/v1/organization/study-programs` tanpa login → daftar prodi tampil
- [ ] Buka MinIO console (port tercantum di laporan agent) → bucket sudah dibuat
- [ ] Request langsung ke port organization-service dengan token valid → 401 `GATEWAY_SIGNATURE_INVALID`

---

## Tahap 2 — Identity Service + OCR KTP (26 Sep – 6 Okt)

```
Kerjakan Tahap 2: F2-M2 (Identity Service, Bagian 4.2) lalu F2-M2b
(OCR KTP, Bagian 4.11). Ikuti docs/AGENT_RULES.md.

F2-M2 penting: NIK terenkripsi AES-GCM + nik_hash, masking NIK untuk
role lain, endpoint /me/completeness, foto profil ke MinIO.
F2-M2b penting:
- OCR Service Python 3.12 + FastAPI, kontrak mengikuti pola fase 1.
- DILARANG model generatif untuk perbaikan gambar.
- DILARANG foto KTP asli; buat generator KTP sintetis + varian
  buram/gelap/silau/miring/resolusi rendah.
- Hasil OCR hanya saran; wajib konfirmasi user.
- OCR_CLOUD_ENABLED=false; tidak ada NIK/gambar di log.
- Jalankan `make ocr-benchmark` dan sertakan docs/ocr-benchmark.md.

e2e: isi biodata manual → completeness true; user A tidak bisa membaca
biodata user B; scan KTP sintetis → confirm → biodata tersimpan; KTP
sangat buram → retake_required.
Commit per milestone. Laporkan lalu BERHENTI.
```

**Cek manual:**
- [ ] Di database identity, kolom `nik` tidak terbaca sebagai angka biasa (terenkripsi)
- [ ] Coba foto KTP **milikmu sendiri** (jelas dan sengaja buram) → field terisi atau diminta foto ulang dengan alasan jelas. Jangan commit fotonya
- [ ] `docker logs` identity & ocr service → tidak ada NIK
- [ ] `docs/ocr-benchmark.md` memenuhi target akurasi PRD 4.11

---

## Tahap 3 — Notification + Admission bagian 1 (7–13 Okt)

```
Kerjakan Tahap 3: F2-M3 (Notification Service, Bagian 4.3) lalu F2-M4
(Admission bagian 1: master data gelombang/jalur/kuota/persyaratan
dokumen, buat pendaftaran, state machine awal, Bagian 4.4). Ikuti
docs/AGENT_RULES.md.

Catatan: tagihan pendaftaran dibuat oleh Finance yang BELUM ada. Untuk
tahap ini, pembuatan tagihan lewat pkg/outbox ke endpoint internal
Finance; biarkan tertahan di outbox dan buktikan lewat test bahwa event
tersimpan dan akan dikirim ulang. Status pendaftaran berhenti di
awaiting_payment.

e2e: pendaftaran di luar periode ditolak (PERIOD_CLOSED); pendaftaran
ganda di gelombang yang sama ditolak (DUPLICATE_APPLICATION); ganti
pilihan prodi saat awaiting_payment berhasil; transisi ilegal ditolak
(INVALID_STATE); notifikasi contoh masuk Mailpit.
Commit per milestone. Laporkan lalu BERHENTI.
```

**Cek manual:**
- [ ] Matikan Mailpit sebentar (`docker stop pmb-mailpit`), picu notifikasi, nyalakan lagi → email tetap terkirim (retry)
- [ ] Login `prodi` → hanya melihat pendaftar prodinya sendiri
- [ ] Nomor pendaftaran berformat `PMB-YYYY-...`

---

## Tahap 4 — Finance bagian 1 (14–20 Okt)

```
Kerjakan Tahap 4: F2-M5 (Finance bagian 1, Bagian 4.5): skema biaya,
tagihan, pembayaran VA dengan MockProvider, webhook idempoten,
endpoint dev simulate-paid (hanya APP_ENV=development), transfer manual
+ verifikasi keuangan, kwitansi PDF. Outbox Admission dari Tahap 3
harus otomatis terkirim sehingga tagihan terbentuk. Tagihan lunas →
Admission ke form_open. Ikuti docs/AGENT_RULES.md.

e2e: buat pendaftaran → tagihan otomatis → bayar VA (simulate-paid) →
status form_open; webhook dikirim dua kali → diproses sekali; webhook
signature salah → 401; transfer manual → keuangan approve → lunas;
user A tidak bisa melihat tagihan user B; nominal selalu integer.
Commit. Laporkan lalu BERHENTI.
```

**Cek manual:**
- [ ] Kwitansi PDF bisa diunduh dan isinya benar
- [ ] Endpoint `simulate-paid` **tidak** tersedia bila `APP_ENV` bukan development
- [ ] Keuangan menolak bukti transfer → user bisa unggah ulang

---

## Tahap 5 — Admission bagian 2 (21–26 Okt)

```
Kerjakan Tahap 5: F2-M6 (Admission bagian 2, Bagian 4.4): upload
dokumen ke MinIO (validasi MIME dari isi file), submit (cek completeness
Identity + dokumen wajib), verifikasi admin per dokumen, alur revisi,
presigned URL 5 menit, export xlsx, endpoint statistik, notifikasi
verifikasi/revisi, tampilan data OCR KTP untuk admin (Bagian 4.11
"Dukungan verifikasi admin"). Ikuti docs/AGENT_RULES.md.

e2e: submit tanpa biodata lengkap → BIODATA_INCOMPLETE; tanpa dokumen
wajib → DOCUMENT_REQUIRED; file .exe dengan ekstensi .pdf → ditolak;
admin minta revisi → user upload ulang → admin verifikasi → verified.
Commit. Laporkan lalu BERHENTI.
```

**Cek manual:**
- [ ] Link dokumen dari admin kedaluwarsa setelah 5 menit
- [ ] Export xlsx pendaftar bisa dibuka di Excel
- [ ] Email revisi & verifikasi masuk Mailpit

---

## Tahap 6 — Exam Service (27 Okt – 2 Nov)

```
Kerjakan Tahap 6: F2-M7 (Exam Service, Bagian 4.6): sesi ujian,
dosen penguji, peserta (manual & auto-assign dari status verified),
kehadiran, kartu ujian PDF dengan foto & QR, jalur berkas tanpa sesi.
Ikuti docs/AGENT_RULES.md.

e2e: pendaftar verified dijadwalkan → status exam_scheduled → kartu
dapat diunduh; pendaftar belum verified tidak bisa dijadwalkan; dosen
hanya melihat sesi yang ditugaskan; kapasitas sesi tidak terlampaui.
Commit. Laporkan lalu BERHENTI.
```

**Cek manual:**
- [ ] Kartu ujian PDF: nama, foto, nomor peserta, jadwal, QR terbaca kamera HP
- [ ] Login dosen lain → sesi dosen pertama tidak terlihat

---

## Tahap 7 — Scoring Service (3–6 Nov)

```
Kerjakan Tahap 7: F2-M8 (Scoring Service, Bagian 4.7): komponen &
bobot per jalur (total 100), input nilai oleh dosen penguji, rekap &
ranking per prodi, export xlsx, flag locked. Ikuti docs/AGENT_RULES.md.

e2e: dosen input nilai peserta sesinya → berhasil; peserta sesi lain →
403; bobot tidak 100 → ditolak; prodi melihat ranking hanya untuk
prodinya; nilai final = rata-rata tertimbang yang benar.
Commit. Laporkan lalu BERHENTI.
```

**Cek manual:**
- [ ] Hitung manual satu nilai akhir → sama dengan hasil sistem
- [ ] Export rekap xlsx sesuai tampilan API

---

## Tahap 8 — Selection Service (7–12 Nov)

```
Kerjakan Tahap 8: F2-M9 (Selection Service, Bagian 4.8): rekomendasi
prodi (validasi kuota), submit → kunci nilai di Scoring, persetujuan
pimpinan (approve/return), publikasi pengumuman oleh admin → results,
update status Admission (passed/failed), permintaan tagihan daftar
ulang ke Finance, notifikasi. Ikuti docs/AGENT_RULES.md.

e2e: rekomendasi melebihi kuota → QUOTA_EXCEEDED; setelah submit,
dosen tidak bisa mengubah nilai → SCORES_LOCKED; pimpinan return →
prodi bisa ubah; sebelum publikasi calon mahasiswa → RESULT_NOT_PUBLISHED;
setelah publikasi → hasil tampil & tagihan daftar ulang terbentuk.
Commit. Laporkan lalu BERHENTI.
```

**Cek manual:**
- [ ] Login `pimpinan` → hanya bisa approve/return, tidak bisa mengedit rekomendasi
- [ ] Email pengumuman masuk Mailpit untuk yang lulus dan tidak lulus

---

## Tahap 9 — Finance bagian 2 (13–18 Nov)

```
Kerjakan Tahap 9: F2-M10 (Finance bagian 2, Bagian 4.5): pengajuan &
persetujuan cicilan (default maks 4 termin, min 25% termin pertama,
jatuh tempo 30 hari), jadwal termin, pembayaran per termin, reminder
H-3 & H-1, penandaan expired, laporan ringkasan & export xlsx untuk
keuangan/pimpinan. Ikuti docs/AGENT_RULES.md.

e2e: cicilan untuk tagihan pendaftaran → INSTALLMENT_NOT_ALLOWED;
ajukan 3 termin → keuangan approve → jumlah termin = total tagihan
(integer, tanpa selisih pembulatan); bayar termin 1 → status
partially_paid; laporan memuat pembayaran tersebut.
Commit. Laporkan lalu BERHENTI.
```

**Cek manual:**
- [ ] Tagihan dengan nominal ganjil (mis. Rp 3.333.334) dibagi 3 termin → totalnya tetap persis sama
- [ ] Reminder bisa dites dengan tanggal jatuh tempo dekat (minta agent jelaskan caranya)

---

## Tahap 10 — Student Service (19–21 Nov)

```
Kerjakan Tahap 10: F2-M11 bagian Student (Bagian 4.9): daftar ulang
(syarat: passed + lunas atau termin 1 dibayar + data tambahan),
tabel students, export untuk admin/prodi/pimpinan. Panggilan ke NIM
Service disiapkan lewat outbox (NIM Service dibuat di tahap berikutnya).
Ikuti docs/AGENT_RULES.md.

e2e: daftar ulang sebelum bayar → PAYMENT_REQUIRED; setelah termin 1
dibayar → daftar ulang berhasil → status reregistered; event penerbitan
NIM tersimpan di outbox.
Commit. Laporkan lalu BERHENTI.
```

**Cek manual:**
- [ ] Yang tidak lulus tidak bisa membuka halaman daftar ulang

---

## Tahap 11 — NIM Service + integrasi akhir (22–30 Nov)

```
Kerjakan Tahap 11: F2-M11 bagian NIM (Bagian 4.9) lalu F2-M12
(integrasi akhir). Ikuti docs/AGENT_RULES.md.

NIM: format default dari PRD, sequence dengan SELECT ... FOR UPDATE,
idempoten per application_id, tambah role mahasiswa di Auth, status
Admission enrolled, notifikasi nim_issued. Outbox dari Tahap 10 harus
terkirim otomatis.

Integrasi akhir: scripts/e2e.sh menjalankan SELURUH alur PRD-fase-2
Bagian 7 dari database kosong; dashboard statistik pimpinan; semua
dokumentasi, Postman, class diagram, dan ERD lengkap; tabel
keterlacakan UC-01 s/d UC-33 semua Selesai; checklist Definition of
Done (Bagian 10) dilaporkan satu per satu.

Tambahkan test konkurensi: 50 permintaan NIM bersamaan → tidak ada NIM
ganda.
Commit per milestone. Laporkan lalu BERHENTI.
```

**Cek manual:**
- [ ] Setelah NIM terbit, logout lalu login lagi → token memuat role `mahasiswa`
- [ ] `docker compose down -v` → `up -d --build` → e2e penuh FAIL: 0
- [ ] Buka setiap `docs/class-diagram-*.md` di GitHub → tampil benar

---

## Prompt untuk perubahan dari pak Kemal (kapan saja)

```
Ada perubahan dari lead:
<tempel pesan pak Kemal atau jelaskan perubahannya>

Langkah:
1. Catat sebagai CR baru di docs/CHANGE_REQUESTS.md.
2. Analisis dampaknya: service, tabel, endpoint, test, dan tahap mana
   yang terdampak, serta apakah mengubah pekerjaan yang sudah selesai.
3. Usulkan perubahan PRD-fase-2.md (tunjukkan bagian yang diubah).
JANGAN mengubah kode dulu. Laporkan lalu BERHENTI menunggu persetujuanku.
```

## Prompt jika agent gagal / ada temuan

```
Temuan saat uji manual Tahap <n>:
- Request: <method + URL + body>
- Hasil: <status + response>
- Seharusnya: <yang benar menurut PRD>
Perbaiki, tambahkan test regresi di unit test dan e2e.sh, commit
"fix(<service>): ...", jalankan ulang semua pemeriksaan, laporkan.
```
