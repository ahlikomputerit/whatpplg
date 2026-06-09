# Setup GitHub & SSH — Panduan Cepat

## 1. Generate SSH Key (sekali aja)

```bash
ssh-keygen -t ed25519 -C "email@github.com"
# Enter → Enter → Enter (accept default, no passphrase)
```

## 2. Add ke GitHub

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

## Catatan

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
