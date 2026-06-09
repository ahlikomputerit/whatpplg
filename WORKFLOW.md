# Workflow Developer — Edit Source → Deploy ke PC Target

## Alur

```
PC Kamu (Developer)                    PC Target (User)
─────────────────                      ────────────────
1. Edit source code
2. docker build
3. docker push          ────────→      4. docker pull
                                       5. docker rm -f wa-gateway
                                       6. docker run ...
```

## Step by Step

### 1. Edit source code

Ubah file `.go` sesuai kebutuhan.

### 2. Build image

```bash
cd /root/whatsapp_baru/whatsmeow-antiban

docker build -f Dockerfile.release \
  -t ghcr.io/ahlikomputerit/whatpplg:latest \
  -t ghcr.io/ahlikomputerit/whatpplg:v1.0.0 .
```

Ganti `ghcr.io/ahlikomputerit/whatpplg` dengan image registry kamu sendiri.

### 3. Push ke registry

```bash
docker push ghcr.io/ahlikomputerit/whatpplg:latest
docker push ghcr.io/ahlikomputerit/whatpplg:v1.0.0
```

### 4. Di PC Target — pull image baru

```bash
docker pull ghcr.io/ahlikomputerit/whatpplg:latest
```

### 5. Di PC Target — recreate container

```bash
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

Session WA tetap aman karena data ada di volume `wa-gateway-data`.

### 6. Verifikasi

```bash
docker ps | grep wa-gateway
docker logs wa-gateway
curl http://localhost:8080/api/v1/health
```

## Biar Cepat — Satu Perintah (PC Target)

Simpan ini sebagai `update.sh` di PC Target:

```bash
#!/bin/bash
docker pull ghcr.io/ahlikomputerit/whatpplg:latest
docker rm -f wa-gateway 2>/dev/null
docker run -d \
  --name wa-gateway \
  --restart unless-stopped \
  -p 8080:8080 \
  -v wa-gateway-data:/app/data \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -e TZ=Asia/Jakarta \
  ghcr.io/ahlikomputerit/whatpplg:latest
echo "✅ Updated"
```

Jalankan:
```bash
chmod +x update.sh
./update.sh
```

## Catatan

| Item | Keterangan |
|------|------------|
| Image publik? | Bisa diatur Public/Private di GHCR |
| Session WA | Aman di volume `wa-gateway-data` |
| Config | File `config.yaml` di luar container, di-mount read-only |
| Port | Sesuaikan `-p 8080:8080` kalau beda port |
| Registry | Ganti `ghcr.io/ahlikomputerit/whatpplg` punya kamu |
