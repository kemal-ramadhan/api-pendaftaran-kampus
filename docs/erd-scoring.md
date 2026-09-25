# Entity Relationship Diagram — Scoring Service (pmb_scoring)

Dokumen ini mendokumentasikan skema relasional tabel pada basis data `pmb_scoring`.

```mermaid
erDiagram
    SCORE_COMPONENTS {
        uuid id PK
        uuid path_id FK "Jalur Seleksi Admission"
        varchar code "written_test, interview, etc"
        varchar name "Nama komponen penilaian"
        numeric weight_percent "Persentase bobot (1 - 100)"
        numeric max_score "Skor maksimal (default 100)"
        boolean is_active "Status komponen aktif"
        timestamptz created_at
        timestamptz updated_at
    }

    SCORES {
        uuid id PK
        uuid application_id FK "Pendaftaran Admission"
        varchar component_code "Kode komponen nilai"
        numeric score "Skor perolehan peserta"
        uuid input_by FK "ID dosen penguji / admin"
        timestamptz input_at "Waktu input nilai"
        text note "Catatan penilaian penguji"
        timestamptz created_at
        timestamptz updated_at
    }

    SCORE_SUMMARIES {
        uuid id PK
        uuid application_id UK "Pendaftaran Admission (Unique)"
        uuid period_id "Gelombang/Periode PMB"
        uuid path_id "Jalur Seleksi PMB"
        varchar study_program_code "Kode Prodi Pilihan 1"
        numeric final_score "Rata-rata tertimbang akhir"
        int rank_in_program "Peringkat dinamis dalam prodi"
        timestamptz computed_at "Waktu kalkulasi terakhir"
        timestamptz created_at
        timestamptz updated_at
    }

    PROGRAM_LOCKS {
        uuid id PK
        uuid period_id "Gelombang/Periode PMB"
        uuid path_id "Jalur Seleksi PMB"
        varchar study_program_code "Kode Program Studi"
        boolean is_locked "Status kunci pengubahan nilai"
        uuid locked_by "ID user yang mengunci"
        timestamptz locked_at "Waktu penguncian"
        timestamptz created_at
        timestamptz updated_at
    }

    SCORE_COMPONENTS ||--o{ SCORES : "defines code & weight"
    SCORES }o--|| SCORE_SUMMARIES : "aggregates into final score"
    PROGRAM_LOCKS ||--o{ SCORE_SUMMARIES : "locks score mutation"
```

### Penjelasan Kunci & Aturan Bisnis
1. **Total Bobot Wajib 100%**: $\sum \text{weight\_percent} = 100.00$ untuk seluruh komponen aktif pada suatu `path_id`.
2. **Keunikan Nilai Peserta**: `UNIQUE(application_id, component_code)` memastikan satu komponen nilai hanya memiliki 1 catatan per peserta.
3. **Penguncian Nilai**: `UNIQUE(period_id, path_id, study_program_code)` pada `program_locks` melindungi seluruh perubahan nilai setelah tahap evaluasi rekomendasi dimulai.
