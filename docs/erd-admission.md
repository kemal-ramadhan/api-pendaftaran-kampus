# Entity Relationship Diagram (ERD) — Admission Service

Dokumen ini memetakan relasi tabel-tabel pada basis data `pmb_admission`.

```mermaid
erDiagram
    PERIODS ||--o{ QUOTAS : "memiliki"
    PERIODS ||--o{ APPLICATIONS : "berisi"
    ADMISSION_PATHS ||--o{ DOCUMENT_REQUIREMENTS : "mensyaratkan"
    ADMISSION_PATHS ||--o{ QUOTAS : "mengalokasikan"
    ADMISSION_PATHS ||--o{ APPLICATIONS : "dipilih"
    APPLICATIONS ||--o{ APPLICATION_DOCUMENTS : "melampirkan"
    APPLICATIONS ||--o{ APPLICATION_STATUS_LOGS : "mencatat"
    APPLICATIONS ||--o{ APPLICATION_CHANGES : "memiliki riwayat"
    APPLICATIONS ||--o{ PATH_CHANGE_REQUESTS : "mengajukan"
    APPLICATIONS ||--o{ WITHDRAWALS : "mengundurkan diri"

    PERIODS {
        serial id PK "ID Periode pendaftaran"
        varchar(100) name "Nama periode / gelombang"
        varchar(20) academic_year "Tahun akademik, cth: 2026/2027"
        timestamptz start_date "Waktu mulai pendaftaran"
        timestamptz end_date "Waktu selesai pendaftaran"
        boolean is_active "Status aktif periode"
        timestamptz created_at "Waktu pembuatan"
        timestamptz updated_at "Waktu perubahan"
    }

    ADMISSION_PATHS {
        serial id PK "ID Jalur masuk"
        varchar(100) name "Nama jalur masuk"
        varchar(20) code UK "Kode jalur masuk (REG, PRES-AK, dll)"
        text description "Deskripsi jalur"
        numeric(12,2) registration_fee "Biaya pendaftaran"
        boolean requires_exam "Wajib tes atau tidak"
        boolean is_active "Status aktif jalur"
        timestamptz created_at "Waktu pembuatan"
        timestamptz updated_at "Waktu perubahan"
    }

    DOCUMENT_REQUIREMENTS {
        uuid id PK "UUID Persyaratan"
        int admission_path_id FK "ID Jalur Masuk"
        varchar(50) document_type "Tipe dokumen: ijazah, rapor, ktp, kk"
        varchar(100) name "Nama berkas persyaratan"
        text description "Petunjuk unggah berkas"
        boolean is_required "Wajib atau opsional"
        int max_size_mb "Batas ukuran berkas (MB)"
        varchar(100) allowed_formats "Format file yang diizinkan"
        timestamptz created_at "Waktu pembuatan"
        timestamptz updated_at "Waktu perubahan"
    }

    QUOTAS {
        uuid id PK "UUID Kuota"
        int period_id FK "ID Periode"
        int admission_path_id FK "ID Jalur"
        uuid study_program_id "ID Program Studi"
        int capacity "Kapasitas daya tampung"
        int filled_count "Jumlah kuota terisi"
        timestamptz created_at "Waktu pembuatan"
        timestamptz updated_at "Waktu perubahan"
    }

    APPLICATIONS {
        uuid id PK "UUID Formulir Pendaftaran"
        varchar(50) registration_no UK "Nomor Registrasi PMB-YYYY-P-XXXXX"
        uuid user_id FK "ID Pengguna Pendaftar"
        int period_id FK "ID Periode"
        int admission_path_id FK "ID Jalur Masuk"
        uuid choice_1_id "Pilihan Prodi 1"
        uuid choice_2_id "Pilihan Prodi 2 (opsional)"
        varchar(30) status "Status pendaftaran: awaiting_payment, withdrawn, dll"
        uuid invoice_id "ID Tagihan Finance (opsional)"
        uuid final_choice_id "Prodi Diterima (opsional)"
        int path_change_count_after_payment "Jumlah pindah jalur setelah bayar"
        uuid previous_application_id FK "ID Pendaftaran Sebelumnya (nullable)"
        timestamptz created_at "Waktu pendaftaran"
        timestamptz updated_at "Waktu perubahan status"
    }

    APPLICATION_DOCUMENTS {
        uuid id PK "UUID Dokumen"
        uuid application_id FK "ID Pendaftaran"
        varchar(50) document_type "Jenis dokumen"
        varchar(255) file_name "Nama berkas asli"
        varchar(500) file_url "URL berkas di MinIO"
        bigint file_size "Ukuran berkas dalam bytes"
        varchar(20) status "Status verifikasi: pending/approved/rejected"
        text notes "Catatan verifikator"
        uuid verified_by "User ID verifikator"
        timestamptz verified_at "Waktu diverifikasi"
        timestamptz created_at "Waktu diunggah"
        timestamptz updated_at "Waktu perubahan"
    }

    APPLICATION_STATUS_LOGS {
        uuid id PK "UUID Log Status"
        uuid application_id FK "ID Pendaftaran"
        varchar(30) from_status "Status asal"
        varchar(30) to_status "Status tujuan"
        uuid changed_by "User ID pengubah status"
        text notes "Keterangan perubahan status"
        timestamptz created_at "Waktu pencatatan"
    }

    APPLICATION_CHANGES {
        uuid id PK "UUID Riwayat Perubahan (Immutable)"
        uuid application_id FK "ID Pendaftaran"
        uuid user_id FK "ID Pengguna"
        varchar(50) change_type "created, choices_updated, path_change_*, withdrawal_*"
        jsonb from_values "Data sebelum perubahan"
        jsonb to_values "Data sesudah perubahan"
        text reason "Alasan perubahan"
        bigint fee_difference "Selisih biaya pendaftaran"
        uuid actor_id "ID pelaku perubahan"
        varchar(50) actor_role "applicant, admin_pmb, system"
        varchar(20) source "applicant, admin, system"
        uuid related_request_id "ID permintaan terkait"
        timestamptz created_at "Waktu pencatatan"
    }

    PATH_CHANGE_REQUESTS {
        uuid id PK "UUID Permohonan Pindah Jalur"
        uuid application_id FK "ID Pendaftaran"
        uuid user_id FK "ID Pengguna"
        int from_period_id "ID Periode Asal"
        int to_period_id "ID Periode Tujuan"
        int from_path_id "ID Jalur Asal"
        int to_path_id "ID Jalur Tujuan"
        uuid from_choice_1_id "Prodi 1 Asal"
        uuid to_choice_1_id "Prodi 1 Tujuan"
        uuid from_choice_2_id "Prodi 2 Asal"
        uuid to_choice_2_id "Prodi 2 Tujuan"
        text reason "Alasan pindah jalur"
        varchar(20) status "pending, approved, rejected, cancelled"
        bigint fee_difference "Selisih biaya pendaftaran"
        uuid reviewed_by "ID Admin Peninjau"
        text review_note "Catatan Peninjauan"
        timestamptz reviewed_at "Waktu Peninjauan"
        timestamptz created_at "Waktu Pengajuan"
    }

    WITHDRAWALS {
        uuid id PK "UUID Pengunduran Diri"
        uuid application_id FK "ID Pendaftaran"
        uuid user_id FK "ID Pengguna"
        varchar(50) stage_at_request "Tahap pendaftaran saat permohonan"
        varchar(50) reason_category "financial, accepted_elsewhere, family, health, etc"
        text reason_text "Uraian alasan"
        varchar(20) status "pending, approved, rejected, cancelled, effective"
        boolean requires_approval "Memerlukan persetujuan admin atau tidak"
        timestamptz requested_at "Waktu diajukan"
        uuid reviewed_by "ID Admin Peninjau"
        text review_note "Catatan persetujuan/penolakan"
        timestamptz reviewed_at "Waktu peninjauan"
        timestamptz effective_at "Waktu resmi berlaku"
        varchar(20) source "applicant, admin"
        varchar(255) attachment_object_key "Lampiran surat jika diajukan admin"
        varchar(100) letter_number "Nomor Surat PMB/UND/YYYY/XXXX"
        varchar(255) letter_object_key "File PDF surat resmi"
        varchar(64) verify_code_hash "Hash verifikasi publik QR"
        timestamptz created_at "Waktu pembuatan"
        timestamptz updated_at "Waktu perubahan"
    }

    OUTBOX_EVENTS {
        uuid id PK "UUID Outbox Event"
        varchar(50) aggregate_type "Tipe agregat: application"
        varchar(100) aggregate_id "ID agregat"
        varchar(100) event_type "Tipe event: invoice.create"
        text payload "Payload JSON event"
        varchar(150) idempotency_key UK "Kunci idempoten"
        varchar(20) status "Status: pending, published, failed"
        int retry_count "Jumlah retry"
        text last_error "Pesan error terakhir"
        timestamptz created_at "Waktu pembuatan event"
        timestamptz processed_at "Waktu selesai diproses"
    }
```
