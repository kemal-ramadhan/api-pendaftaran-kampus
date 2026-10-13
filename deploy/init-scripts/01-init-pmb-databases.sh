#!/bin/bash
set -e

echo "Menjalankan inisialisasi basis data PMB (Konsolidasi 2 Database: pmb_auth, pmb_app)..."

databases=(
    "pmb_auth"
    "pmb_app"
)

for db in "${databases[@]}"; do
    echo "Memeriksa database $db..."
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
        SELECT 'CREATE DATABASE $db'
        WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$db')\gexec
        GRANT ALL PRIVILEGES ON DATABASE $db TO $POSTGRES_USER;
EOSQL
done

echo "Seluruh database PMB (pmb_auth, pmb_app) berhasil disiapkan!"
