# Entity Relationship Diagram (ERD) — Finance Service

Dokumen ini memetakan relasi tabel-tabel pada basis data `pmb_finance`.

```mermaid
erDiagram
    INVOICES ||--o{ PAYMENTS : "memiliki"
    INVOICES ||--o{ INSTALLMENT_REQUESTS : "diajukan"
    INVOICES ||--o{ INSTALLMENTS : "terbagi_menjadi"
    INSTALLMENTS ||--o{ PAYMENTS : "dibayar_lewat"
    PAYMENTS ||--o| RECEIPTS : "menerbitkan"
    PAYMENTS ||--o| REFUND_REVIEWS : "berpotensi_menjadi"
    INVOICES ||--o{ REFUND_REVIEWS : "direview_pada"
    FEE_SCHEMES }o--|| PERIODS : "berlaku_pada"

    FEE_SCHEMES {
        uuid id PK "UUID Skema Biaya"
        int period_id "ID Periode pendaftaran"
        int admission_path_id "ID Jalur masuk"
        varchar(30) study_program_code "Kode Program Studi (opsional)"
        varchar(50) fee_type "registration / reregistration"
        bigint amount "Nominal uang bulat Rupiah"
        boolean installment_allowed "Apakah boleh dicicil"
        int max_installments "Maksimal termin cicilan"
        int min_first_payment_percent "Minimal persentase DP (%)"
        timestamptz created_at "Waktu pembuatan"
        timestamptz updated_at "Waktu perubahan"
    }

    INVOICES {
        uuid id PK "UUID Tagihan"
        varchar(50) invoice_number UK "Format: INV-YYYYMMDD-XXXXX"
        uuid user_id "ID Pengguna / Calon Mahasiswa"
        uuid application_id "ID Pendaftaran (opsional)"
        varchar(50) fee_type "registration / reregistration"
        text description "Keterangan tagihan"
        bigint amount "Total nominal tagihan (Rupiah integer)"
        bigint paid_amount "Jumlah yang sudah terbayar (Rupiah)"
        varchar(30) status "unpaid, paid, partially_paid, expired, cancelled"
        timestamptz due_date "Batas akhir pembayaran"
        varchar(30) payment_scheme "full / installment"
        timestamptz created_at "Waktu pembuatan"
        timestamptz updated_at "Waktu perubahan"
    }

    INSTALLMENT_REQUESTS {
        uuid id PK "UUID Pengajuan Cicilan"
        uuid invoice_id FK "Relasi ke tagihan"
        uuid user_id "ID Mahasiswa Pemohon"
        int num_terms "Jumlah termin yang diajukan (2-4)"
        text reason "Alasan permohonan cicilan"
        varchar(30) status "pending, approved, rejected"
        text notes "Catatan dari peninjau bagian keuangan"
        uuid reviewed_by "ID Staf keuangan yang meninjau"
        timestamptz reviewed_at "Waktu peninjauan"
        timestamptz created_at "Waktu pengajuan"
        timestamptz updated_at "Waktu perubahan"
    }

    INSTALLMENTS {
        uuid id PK "UUID Termin Cicilan"
        uuid invoice_id FK "Relasi ke tagihan"
        int term_number "Nomor urut termin (1, 2, ...)"
        bigint amount "Nominal termin dalam Rupiah integer"
        timestamptz due_date "Batas akhir pembayaran termin (+30 hari per termin)"
        varchar(30) status "unpaid, paid, expired"
        timestamptz paid_at "Waktu pelunasan termin"
        timestamptz created_at "Waktu pembentukan jadwal termin"
        timestamptz updated_at "Waktu perubahan"
    }

    PAYMENTS {
        uuid id PK "UUID Transaksi Pembayaran"
        uuid invoice_id FK "Relasi ke tagihan"
        uuid installment_id FK "Relasi ke termin cicilan (opsional)"
        varchar(30) method "virtual_account / manual_transfer"
        varchar(50) provider "mock / midtrans / xendit / manual"
        varchar(100) provider_ref UK "ID referensi transaksi dari provider"
        varchar(50) va_number "Nomor Virtual Account"
        bigint amount "Jumlah uang yang dibayarkan"
        varchar(30) status "pending, success, failed, expired, awaiting_verification, rejected"
        varchar(255) proof_object_key "Kunci objek berkas bukti transfer di MinIO"
        varchar(500) proof_url "URL akses presigned bukti transfer"
        text notes "Catatan verifikasi staf keuangan"
        uuid verified_by "ID Staf yang memverifikasi manual"
        timestamptz verified_at "Waktu verifikasi transfer manual"
        timestamptz paid_at "Waktu pembayaran lunas"
        timestamptz created_at "Waktu transaksi dibuat"
        timestamptz updated_at "Waktu transaksi diperbarui"
    }

    RECEIPTS {
        uuid id PK "UUID Kwitansi"
        uuid payment_id FK "Relasi ke transaksi pembayaran"
        uuid invoice_id FK "Relasi ke tagihan"
        varchar(50) receipt_number UK "Format: KW-YYYYMMDD-XXXXX"
        varchar(255) pdf_object_key "Kunci berkas PDF kwitansi di MinIO"
        varchar(500) pdf_url "URL presigned kwitansi PDF"
        bigint amount "Nominal yang tertera pada kwitansi"
        timestamptz created_at "Waktu penerbitan kwitansi"
    }

    PAYMENT_WEBHOOK_LOGS {
        uuid id PK "UUID Log Webhook"
        varchar(50) provider "Nama payment gateway"
        varchar(100) provider_ref "ID referensi event webhook"
        text payload "Payload JSON mentah dari provider"
        varchar(255) signature "Signature HMAC header webhook"
        varchar(30) status "processed, rejected, duplicate"
        text error_message "Pesan kegagalan pemrosesan"
        timestamptz created_at "Waktu penerimaan webhook"
    }

    REFUND_REVIEWS {
        uuid id PK "UUID Antrean Peninjauan Pengembalian Dana"
        uuid payment_id UK "ID Pembayaran terkait (1-to-1)"
        uuid invoice_id "ID Tagihan terkait"
        uuid application_id "ID Pendaftaran yang mengundurkan diri"
        bigint amount "Nominal dana yang ditinjau untuk refund"
        varchar(30) status "pending, approved, rejected, processed"
        text notes "Catatan review atau nomor referensi transfer refund"
        uuid reviewed_by "ID Staf keuangan yang memproses"
        timestamptz reviewed_at "Waktu pemrosesan review"
        timestamptz created_at "Waktu masuk antrean"
        timestamptz updated_at "Waktu perubahan data"
    }
```

## Penjelasan Tabel & Integritas Data (CR-007)

1. **`refund_reviews`**:
   - Dibuat secara otomatis ketika suatu pendaftaran mahasiswa disetujui pengunduran dirinya (`withdrawn`) melalui Admission Service.
   - Seluruh transaksi pembayaran yang berstatus lunas (`success`) pada pendaftaran tersebut akan didaftarkan ke tabel `refund_reviews` dengan status awal `pending`.
   - Staf keuangan dapat meninjau, menyetujui, mencairkan, atau menolak pengembalian dana melalui antarmuka Finance Service.
   - Kolom `payment_id` bersifat unik (`UNIQUE`) guna mencegah pendaftaran duplikat untuk transaksi pembayaran yang sama.


