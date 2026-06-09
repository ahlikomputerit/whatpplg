# Panduan Docker — WA Gateway Service

Dokumentasi lengkap build, publish, install, dan operasional sehari-hari.

---

## Daftar Isi

1. [Arsitektur Distribusi](#1-arsitektur-distribusi)
2. [Setup Docker di PC Kamu](#2-setup-docker-di-pc-kamu)
3. [Build & Publish ke Registry](#3-build--publish-ke-registry)
4. [Install di PC Target (User)](#4-install-di-pc-target-user)
5. [Konfigurasi](#5-konfigurasi)
6. [Operasional Sehari-hari](#6-operasional-sehari-hari)
7. [Update Image](#7-update-image)
8. [FAQ](#8-faq)

---

## 1. Arsitektur Distribusi

```
┌─────────────────────────────────────────────────────────────┐
│                    PC Kamu (Developer)                       │
│                                                             │
│  Source Code + whatsmeow embedded                           │
│        │                                                    │
│        ▼                                                    │
│  docker build -t wa-gateway .                               │
│        │                                                    │
│        ▼                                                    │
│  docker push ghcr.io/ahlikomputerit/whatpplg:latest             │
│        │                                                    │
│        └──────────┬──────────┬──────────┬── ...             │
│                   │          │          │                   │
└───────────────────┼──────────┼──────────┼───────────────────┘
                    │          │          │
                    ▼          ▼          ▼
            ┌──────────────────────────────┐
            │     PC Target (User)          │
            │                               │
            │  curl ... install.sh | bash   │
            │  docker pull [image]          │
            │  docker run [container]       │
            │                               │
            │  ✅ NO source code di sini    │
            │  ✅ Cuma Docker + config      │
            └──────────────────────────────┘
```

**Pemisahan peran:**

| Peran | Tugas | Akses Source Code |
|-------|-------|-------------------|
| **Developer** (kamu) | build, publish, update | ✅ Punya full source |
| **User** (PC target) | install, run, pakai API | ❌ Tidak punya source |

---

## 2. Setup Docker di PC Kamu

### Install Docker

```bash
# Linux (Ubuntu/Debian)
sudo apt update && sudo apt install docker.io docker-compose-v2
sudo systemctl enable --now docker

# Arch Linux
sudo pacman -S docker docker-compose

# Fedora
sudo dnf install docker docker-compose
```

### Login ke GitHub Container Registry

```bash
# Buat token dulu di https://github.com/settings/tokens
# Pilih scope: write:packages, read:packages

echo "TOKEN_AND A" | docker login ghcr.io -u USERNAME --password-stdin
```

### Cek

```bash
docker info
# Output: Server Version: ... (harusnya jalan)
```

---

## 3. Build & Publish ke Registry

### Build Image

```bash
cd whatsmeow-antiban

# Build untuk registry (production)
docker build -f Dockerfile.release -t ghcr.io/ahlikomputerit/whatpplg:latest .

# Build dengan tag version
docker build -f Dockerfile.release \
  -t ghcr.io/ahlikomputerit/whatpplg:latest \
  -t ghcr.io/ahlikomputerit/whatpplg:v1.0.0 .
```

### Push ke Registry

```bash
# Push semua tag
docker push ghcr.io/ahlikomputerit/whatpplg:latest
docker push ghcr.io/ahlikomputerit/whatpplg:v1.0.0
```

### Atau Pakai Script Otomatis

```bash
# Build + push dalam satu perintah
./publish.sh
# Atau specify image name
./publish.sh ghcr.io/ahlikomputerit/whatpplg v1.0.0
```

### Catatan Penting

| Item | Keterangan |
|------|------------|
| Image publik? | Bisa public atau private di GHCR |
| Ukuran image | ~25 MB (Alpine + Go binary) |
| Source di image? | ❌ Tidak. Hanya compiled binary |
| Session WA aman? | ✅ Tersimpan di volume Docker |
| whatsmeow library? | ✅ Ter-compile di binary (CGO_ENABLED=0) |

---

## 4. Install di PC Target (User)

### Prasyarat

- Docker sudah terinstall
- Internet (untuk pull image pertama kali)

### Opsi 1: One-Liner Install (Rekomendasi)

```bash
curl -sL https://raw.githubusercontent.com/ahlikomputerit/whatpplg/main/install.sh | bash
```

Script otomatis:
1. Generate `config.yaml` + API key random
2. Pull image dari registry
3. Jalankan container `wa-gateway`
4. Print API endpoint + key

**Dengan opsi:**

```bash
curl -sL https://raw.githubusercontent.com/ahlikomputerit/whatpplg/main/install.sh | \
  bash -s -- --port 8080 --dir /opt/wa-gateway
```

### Opsi 2: Manual dengan docker run

```bash
# 1. Buat config.yaml (lihat bagian Konfigurasi)

# 2. Pull image
docker pull ghcr.io/ahlikomputerit/whatpplg:latest

# 3. Jalankan
docker run -d \
  --name wa-gateway \
  --restart unless-stopped \
  -p 8080:8080 \
  -v wa-gateway-data:/app/data \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -e TZ=Asia/Jakarta \
  ghcr.io/ahlikomputerit/whatpplg:latest
```

### Opsi 3: Docker Compose

```bash
# 1. Buat folder project
mkdir wa-gateway && cd wa-gateway

# 2. Download docker-compose.yml
curl -sL -o docker-compose.yml \
  https://raw.githubusercontent.com/ahlikomputerit/whatpplg/main/docker-compose.yml

# 3. Buat config.yaml (lihat bagian Konfigurasi)
# 4. Buat folder templates (optional)

# 5. Jalankan
docker compose up -d
```

### QR Code — Login WhatsApp Pertama Kali

```bash
# Lihat log untuk QR code
docker logs -f wa-gateway

# Output:
# === QR CODE WhatsApp ===
# Buka WhatsApp > Settings > Linked Devices > Link a Device
# ██████████████ ...
#
# QR code akan muncul sebagai ASCII art.
# Scan dengan HP > WhatsApp > Linked Devices

# Kalau QR tidak bisa discan dari terminal,
# buka link URL yang tercetak di bawah QR.
```

Setelah scan, session tersimpan di volume `wa-gateway-data`.  
Restart container langsung connect tanpa QR ulang.

---

## 5. Konfigurasi

### config.yaml

```yaml
server:
  port: 8080                          # Port HTTP API
  api_key: "wa-abc123..."             # API Key untuk autentikasi

whatsapp:
  db_path: "/app/data/wa_session.db"  # Path session WA (jangan diubah)
  preset: "moderate"                  # conservative | moderate | aggressive | high-volume
  config:
    enable_typo_injection: true
    enable_zero_width: true
    enable_punctuation_vary: true

sources:
  - name: "default"
    mode: "api"
    api_key: "wa-abc123..."           # API Key untuk source ini

queue:
  type: "memory"                      # memory | redis
  max_size: 10000

templates:
  - name: "laporan-pelanggaran"
    body: |
      Assalamu'alaikum Wr. Wb.
      
      Yth. Bpk/Ibu {nama_ortu}
      
      Diberitahukan bahwa {nama_siswa} kelas {kelas}
      telah melakukan pelanggaran: {deskripsi}
```

### Environment Variables

| Variable | Default | Fungsi |
|----------|---------|--------|
| `TZ` | `Asia/Jakarta` | Timezone container |
| `PORT` | `8080` | Port yang di-expose (docker compose) |
| `CONFIG_FILE` | `./config.yaml` | Path config file |
| `PG_PASSWORD` | `changeme123` | Password PostgreSQL |

---

## 6. Operasional Sehari-hari

### Cek Status

```bash
# Status container
docker ps | grep wa-gateway

# Log real-time
docker logs -f wa-gateway

# Health check API
curl http://localhost:8080/api/v1/health
```

### Kirim Pesan via API

```bash
# Dapatkan API Key dari config.yaml
API_KEY="wa-abc123..."

# Kirim satu pesan
curl -X POST http://localhost:8080/api/v1/send \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "62812xxxxxx",
    "message": "Assalamualaikum, ini pesan test"
  }'

# Kirim dengan template
curl -X POST http://localhost:8080/api/v1/send-template \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "62812xxxxxx",
    "template": "laporan-pelanggaran",
    "data": {
      "nama_ortu": "Budi",
      "nama_siswa": "Andi",
      "kelas": "X-A",
      "deskripsi": "Terlambat 15 menit"
    }
  }'
```

### Restart

```bash
# Restart container (session WA tetap aman)
docker restart wa-gateway

# Atau pakai docker compose
docker compose restart
```

### Berhenti

```bash
# Stop container (data tetap aman di volume)
docker stop wa-gateway

# Start lagi
docker start wa-gateway

# Hapus container (volume tetap aman)
docker rm wa-gateway
```

### Reset Total (termasuk session WA)

```bash
# Hapus container + volume (session WA hilang!)
docker rm -f wa-gateway
docker volume rm wa-gateway-data

# Install ulang
curl -sL ...install.sh | bash
```

### Backup & Restore

```bash
# Backup session WA
docker run --rm -v wa-gateway-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/wa-session-backup.tar.gz -C /data .

# Restore
docker run --rm -v wa-gateway-data:/data -v $(pwd):/backup alpine \
  tar xzf /backup/wa-session-backup.tar.gz -C /data
```

---

## 7. Update Image

### Developer (Kamu)

```bash
# 1. Update source code
# 2. Rebuild
docker build -f Dockerfile.release \
  -t ghcr.io/ahlikomputerit/whatpplg:latest \
  -t ghcr.io/ahlikomputerit/whatpplg:v1.1.0 .

# 3. Push
docker push ghcr.io/ahlikomputerit/whatpplg:latest
docker push ghcr.io/ahlikomputerit/whatpplg:v1.1.0
```

### User (PC Target)

```bash
# Pull image terbaru
docker pull ghcr.io/ahlikomputerit/whatpplg:latest

# Recreate container (volume tetap, session aman)
docker rm -f wa-gateway
docker run -d \
  --name wa-gateway \
  --restart unless-stopped \
  -p 8080:8080 \
  -v wa-gateway-data:/app/data \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -e TZ=Asia/Jakarta \
  ghcr.io/ahlikomputerit/whatpplg:latest
```

Atau dengan docker compose:

```bash
docker compose pull
docker compose up -d
```

---

## 8. FAQ

### Q: Apakah source code saya aman?

**Ya.** Image Docker hanya berisi compiled Go binary.  
Tidak ada file `.go` di dalam container.  
Untuk mengakses source, harus clone repo GitHub kamu yang private.

### Q: Session WA hilang setelah update?

**Tidak.** Session tersimpan di volume `wa-gateway-data`.  
Selama volume tidak dihapus, session aman.

### Q: User perlu install Go?

**Tidak perlu.** Cuma butuh Docker.

### Q: Kalau container mati, data WA hilang?

**Tidak.** Data WA (session + SQLite) di volume Docker.  
Container bisa dihapus, data tetap aman. Tinggal `docker run` lagi.

### Q: Bisa ganti preset?

**Bisa.** Edit `config.yaml` → `whatsapp.preset` → `restart`.  
Container akan baca ulang config saat restart.

### Q: Kalau kena ban WhatsApp?

Tenang, anti-ban sudah aktif di middleware.  
Kalau tetap kena ban:
1. `docker stop wa-gateway`
2. Ganti preset ke `conservative`
3. `docker start wa-gateway`
4. Biarkan beberapa hari sampai recovery phase selesai

### Q: Bisa kirim gambar/file?

Bisa. Nanti endpoint `/api/v1/send` support media URL.  
(Nanti akan ditambahkan ke source code jika diperlukan)

### Q: Berapa maksimal pesan per hari?

Tergantung preset:

| Preset | Max/hari | Cocok |
|--------|----------|-------|
| conservative | 150 | Sangat aman |
| moderate | 400 | Normal (rekomendasi) |
| aggressive | 800 | Agresif |
| high-volume | 2000 | Broadcast |

---

## Referensi

| File | Fungsi |
|------|--------|
| `Dockerfile` | Build image (development, dari source) |
| `Dockerfile.release` | Build image (production, untuk registry) |
| `docker-compose.yml` | Service orchestration |
| `docker-compose.dev.yml` | Development hot-reload |
| `config.yaml` | Konfigurasi gateway |
| `install.sh` | One-liner install script |
| `publish.sh` | Build & push ke registry |
| `Makefile` | Shortcut commands |
| `DESAIN_WA_GATEWAY.md` | Desain arsitektur lengkap |
