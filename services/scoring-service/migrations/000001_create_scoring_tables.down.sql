-- Migrasi down: Hapus tabel skema scoring service
DROP TABLE IF EXISTS program_locks CASCADE;
DROP TABLE IF EXISTS score_summaries CASCADE;
DROP TABLE IF EXISTS scores CASCADE;
DROP TABLE IF EXISTS score_components CASCADE;
