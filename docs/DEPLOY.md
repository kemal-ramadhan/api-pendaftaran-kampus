# Panduan Deployment PMB Backend di Server Produksi (Linux VPS)

Dokumen ini memuat langkah demi langkah penyebaran (deployment) PMB Backend ke VPS Ubuntu/Debian menggunakan Docker Compose sesuai standar **CR-013**.

---

## 1. Kebutuhan Sistem Minimum & Rekomendasi

| Komponen | Spesifikasi Minimum | Rekomendasi Beban Puncak (Pendaftaran Buka) |
| :--- | :--- | :--- |
| **CPU** | 2 vCPU | 4–8 vCPU |
| **RAM** | 4 GB | 8–16 GB |
| **Penyimpanan** | 40 GB NVMe / SSD | 100+ GB NVMe |
| **Sistem Operasi** | Ubuntu 22.04 LTS / 24.04 LTS | Ubuntu 24.04 LTS x86_64 |
| **Docker** | Engine v24.0+ | Docker Engine 26+ & Compose Plugin v2+ |

---

## 2. Persiapan Awal Server VPS

```bash
# Update sistem
sudo apt update && sudo apt upgrade -y

# Instalasi paket penting
sudo apt install -y curl git ufw jq fail2ban

# Pasang Docker resmi
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Konfigurasi Firewall UFW
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp      # SSH
sudo ufw allow 80/tcp      # HTTP
sudo ufw allow 443/tcp     # HTTPS
sudo ufw enable
```

---

## 3. Clone Repository & Setup Environment

```bash
git clone https://github.com/pkm-asia/pmb-backend.git /opt/pmb-backend
cd /opt/pmb-backend

# Salin konfigurasi environment produksi
cp .env.example .env
nano .env
```

Pastikan variabel penting diisi:
- `APP_ENV=production`
- `GATEWAY_SHARED_SECRET` & `INTERNAL_TOKEN` (gunakan string acak minimal 32 karakter)
- `JWT_ACCESS_SECRET` & `JWT_REFRESH_SECRET`
- `DB_PASSWORD`, `REDIS_PASSWORD`, `MINIO_SECRET_KEY`
- `PAYMENT_PROVIDER=duitku`, `DUITKU_ENV=production`, `DUITKU_MERCHANT_CODE`, `DUITKU_API_KEY`
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`

---

## 4. Konfigurasi Reverse Proxy Nginx & SSL Let's Encrypt

Pasang Nginx di host utama untuk meneruskan trafik ke container `gateway` (port `8000`):

```nginx
# /etc/nginx/sites-available/pmb.kampus.ac.id
server {
    server_name pmb.kampus.ac.id;

    client_max_body_size 15M;

    location / {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 60s;
        proxy_connect_timeout 15s;
    }
}
```

Aktifkan sertifikat SSL:
```bash
sudo apt install -y certbot python3-certbot-nginx
sudo ln -s /etc/nginx/sites-available/pmb.kampus.ac.id /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d pmb.kampus.ac.id
```

---

## 5. Menjalankan Docker Compose Produksi

Jalankan container produksi:
```bash
docker compose -f deploy/docker-compose.prod.yml up -d --build
```

Periksa status container:
```bash
docker compose -f deploy/docker-compose.prod.yml ps
```

---

## 6. Otomatisasi Pencadangan Harian (Automated Backup)

Pasang cronjob untuk mencadangkan database PostgreSQL secara berkala:

```bash
sudo nano /usr/local/bin/pmb-backup.sh
```

Isi skrip:
```bash
#!/usr/bin/env bash
set -e
BACKUP_DIR="/var/backups/pmb"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
mkdir -p "$BACKUP_DIR"

# Dump database PostgreSQL dari container
docker exec pmb-postgres pg_dumpall -U pmb_user | gzip > "$BACKUP_DIR/pmb_full_$TIMESTAMP.sql.gz"

# Hapus backup lebih lama dari 14 hari
find "$BACKUP_DIR" -type f -name "*.sql.gz" -mtime +14 -exec rm {} +
echo "[$TIMESTAMP] Pencadangan PMB database selesai."
```

Beri izin eksekusi dan daftarkan ke crontab:
```bash
sudo chmod +x /usr/local/bin/pmb-backup.sh
(crontab -l 2>/dev/null; echo "0 2 * * * /usr/local/bin/pmb-backup.sh >> /var/log/pmb-backup.log 2>&1") | crontab -
```

---

## 7. Pemantauan & Pengecekan Kesehatan (Monitoring & Health Check)

- Pengecekan Gateway Health: `curl -I https://pmb.kampus.ac.id/health`
- Memeriksa Log Container:
  ```bash
  docker compose -f deploy/docker-compose.prod.yml logs -f --tail=100 gateway
  ```
