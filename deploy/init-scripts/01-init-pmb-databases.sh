#!/bin/bash
set -e

echo "Menjalankan inisialisasi basis data multi-service PMB..."

databases=(
    "pmb_organization"
    "pmb_identity"
    "pmb_notification"
    "pmb_admission"
    "pmb_finance"
    "pmb_exam"
    "pmb_scoring"
    "pmb_selection"
    "pmb_student"
    "pmb_nim"
)

for db in "${databases[@]}"; do
    echo "Memeriksa database $db..."
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
        SELECT 'CREATE DATABASE $db'
        WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$db')\gexec
        GRANT ALL PRIVILEGES ON DATABASE $db TO $POSTGRES_USER;
EOSQL
done

echo "Seluruh database PMB Fase 2 berhasil disiapkan!"
