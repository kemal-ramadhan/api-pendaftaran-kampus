-- Migrasi down: Hapus tabel skema selection service
DROP TABLE IF EXISTS results CASCADE;
DROP TABLE IF EXISTS announcements CASCADE;
DROP TABLE IF EXISTS recommendation_items CASCADE;
DROP TABLE IF EXISTS recommendations CASCADE;
