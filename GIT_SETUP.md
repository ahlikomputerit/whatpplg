# Setup GitHub & SSH — Panduan Cepat

## 1. Generate SSH Key (sekali aja)

```bash
ssh-keygen -t ed25519 -C "email@github.com"
# Enter → Enter → Enter (accept default, no passphrase)
```

## 2. Add SSH Key ke GitHub

```bash
cat ~/.ssh/id_ed25519.pub
# Copy output, buka https://github.com/settings/keys
# Klik "New SSH Key", paste, simpan
```

## 3. Init Repo Baru (dari project existing)

```bash
cd /path/to/project
git init
git branch -m main
git remote add origin git@github.com:USER/REPO.git
git add -A
git commit -m "Initial commit"
git push -u origin main
```

## 4. Push Perubahan

```bash
git add -A
git commit -m "pesan perubahan"
git push
```

## 5. Clone Repo

```bash
git clone git@github.com:USER/REPO.git
```

---

# Docker Image — Build & Push ke GHCR

## 1. Generate Token GitHub

Buka https://github.com/settings/tokens → **Generate new token (classic)**
Centang: **write:packages**, **read:packages** → Generate → Copy token

## 2. Login ke GHCR

```bash
echo "TOKEN_ANDA" | docker login ghcr.io -u USERNAME --password-stdin
```

## 3. Build & Push Image

```bash
docker build -f Dockerfile.release -t ghcr.io/USERNAME/REPO:latest .
docker push ghcr.io/USERNAME/REPO:latest
```

## 4. Bikin Image Public (biar PC target bisa pull tanpa login)

1. Buka https://github.com/users/USERNAME/packages/container/REPO
2. Klik **Package settings** (gear icon)
3. **Change visibility** → **Public** → konfirmasi

## 5. Pull & Run di PC Target

```bash
docker pull ghcr.io/USERNAME/REPO:latest

docker rm -f wa-gateway

docker run -d \
  --name wa-gateway \
  --restart unless-stopped \
  -p 8080:8080 \
  -v wa-gateway-data:/app/data \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  ghcr.io/USERNAME/REPO:latest
```

---

# Catatan Perintah

| Perintah | Fungsi |
|----------|--------|
| `ssh-keygen -t ed25519 -C "email"` | Generate SSH key |
| `cat ~/.ssh/id_ed25519.pub` | Lihat public key |
| `git init` | Init repo lokal |
| `git remote add origin git@github.com:USER/REPO.git` | Tambah remote |
| `git add -A` | Stage semua perubahan |
| `git commit -m "msg"` | Commit |
| `git push` | Push ke GitHub |
| `git pull` | Tarik perubahan terbaru |
| `git status` | Cek status file |
| `git log --oneline` | Lihat history commit |
| `docker login ghcr.io` | Login ke GHCR (butuh token) |
| `docker build -f Dockerfile.release -t IMAGE:tag .` | Build image |
| `docker push IMAGE:tag` | Push ke registry |
| `docker pull IMAGE:tag` | Pull dari registry |
