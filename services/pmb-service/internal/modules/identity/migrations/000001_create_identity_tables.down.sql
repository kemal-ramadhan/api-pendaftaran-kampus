-- Migration down: drop identity tables
DROP TABLE IF EXISTS ktp_scans;
DROP TABLE IF EXISTS education_histories;
DROP TABLE IF EXISTS guardians;
DROP TABLE IF EXISTS addresses;
DROP TABLE IF EXISTS persons;
