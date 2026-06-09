# whatsmeow-antiban

[![Go Reference](https://pkg.go.dev/badge/github.com/ahlikomputerit/whatpplg.svg)](https://pkg.go.dev/github.com/ahlikomputerit/whatpplg)
[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8)](https://go.dev)
[![Coverage](https://img.shields.io/badge/coverage-84.5%25-green)]()
[![Tests](https://img.shields.io/badge/tests-200%2B-blue)]()

Anti-ban protection untuk WhatsApp messaging via [whatsmeow](https://github.com/tulir/whatsmeow) (Go).

**whatsmeow-antiban** adalah extension dari **whatsmeow** — Drop-in middleware yang menambahkan
perilaku seperti manusia (human-like behavior) untuk mengurangi risiko pemblokiran akun WhatsApp.
Implementasi ini adalah padanan Go dari [baileys-antiban](https://github.com/) (TypeScript).

> ✅ **Status:** Telah diuji live dengan WhatsApp Web. Pairing QR, autentikasi, pengiriman `!ping`/`!stats` berhasil.

## Cara Kerja

Setiap pesan keluar melewati pipeline ini:

```
BeforeSend → WarmUp → Scheduler → RateLimiter → CircuitBreaker → TimelockGuard → ContactGraph → delay → Send
                ↓
          AfterSend → Record metrics
```

1. **WarmUp** — cek limit harian (hari ke-1: 20 pesan, hari ke-7: 680 pesan)
2. **Scheduler** — cek apakah jam aktif (misal: 08:00–22:00)
3. **RateLimiter** — Gaussian jitter delay, batas per menit/jam/hari, burst allowance
4. **CircuitBreaker** — blokir JID yang sering gagal
5. **TimelockGuard** — blokir kontak baru saat kena error 463
6. **ContactGraph** — batasi kontak stranger, lurk period grup

## Fitur Lengkap

| Modul | Fungsi |
|-------|--------|
| **RateLimiter** | Gaussian jitter delay, batas per minute/hour/day, burst allowance, deteksi pesan identik |
| **WarmUp** | Limit harian progresif (hari 1→target), reset jika inaktif |
| **HealthMonitor** | Score 0–100, risk level low/medium/high/critical, auto-pause |
| **CircuitBreaker** | Per-JID failure tracking (closed→open→half-open), auto-recovery |
| **TimelockGuard** | Tangani error 463 reachout timelock, whitelist known chats |
| **ContactGraph** | State machine stranger→handshake→known, group lurk period |
| **ContentVariator** | Typo injection, zero-width chars, emoji padding, variasi punctuation |
| **DeviceFingerprint** | Acak app version, OS, device model |
| **Scheduler** | Jam aktif, weekend factor, peak hour boost |
| **ProxyRotator** | Round-robin/random/LRU/weighted, auto-failover |
| **BanRecovery** | Fase: paused→recovering→ramping→graduated |
| **ReconnectThrottle** | Ramp bertahap setelah reconnect |
| **RetryTracker** | Klasifikasi alasan retry, deteksi spiral |
| **DeliveryTracker** | Monitoring rasio terkirim vs diterima |
| **GroupGuard** | Rate limit per operasi grup (add/remove/create/invite) |
| **LidResolver** | LRU cache mapping LID↔PN JID |
| **StateManager** | Persist state ke JSON file, auto-save |
| **Presets** | Conservative, Moderate, Aggressive, High-Volume |

## Persyaratan

- Go 1.25+
- whatsmeow sudah termasuk di folder `whatsmeow/` (tidak perlu install terpisah)

## Instalasi

Proyek ini sudah fully self-contained. Cukup clone atau copy folder:

```bash
git clone <repo-url> whatsmeow-antiban
cd whatsmeow-antiban
go build ./...
```

Atau sebagai dependency:

```bash
go get github.com/ahlikomputerit/whatpplg
```

## Uji Coba (Tanpa WhatsApp)

Jalankan demo untuk melihat semua fitur anti-ban:

```bash
go run ./example/demo/
```

Output demo akan menunjukkan:
- Perbedaan antar preset
- Cara kerja rate limiter, warmup, health monitor
- Variasi konten, device fingerprint
- Circuit breaker, ban recovery
- Scheduler dan orchestrator

## Quick Start (Dengan WhatsApp)

### 1. Persiapan Database

Buat file SQLite untuk menyimpan sesi:

```go
container, err := sqlstore.New(context.Background(), "sqlite3",
    "file:whatsmeow.db?_foreign_keys=on", log.Sub("DB"))
```

### 2. Buat Client

```go
device, _ := container.GetFirstDevice(context.Background())
client := whatsmeow.NewClient(device, log.Sub("WA"))
```

### 3. Wrap dengan AntiBan

```go
abc := antiban.WrapClient(client, antiban.PresetModerate)
```

### 4. Start & Connect

```go
abc.Start(context.Background())
client.Connect()
// Scan QR code untuk login pertama kali
```

### 5. Kirim Pesan

```go
resp, err := abc.SendMessage(ctx, toJID, &waE2E.Message{
    Conversation: proto.String("Halo!"),
})
```

### Contoh Lengkap

Lihat `example/main.go` untuk contoh lengkap dengan QR login dan event handling.

## Verifikasi

Proyek telah diuji **live dengan WhatsApp Web**:

| Langkah | Status |
|---------|--------|
| Pairing QR Code | ✅ QR muncul di terminal (ASCII) & URL |
| Autentikasi | ✅ `Successfully authenticated` |
| Kirim `!ping` | ✅ Bot balas `Pong!` |
| Kirim `!stats` | ✅ Statistik anti-ban ditampilkan |
| Sesi tersimpan | ✅ Login ulang tanpa QR |
| Demo tanpa WA | ✅ `go run ./example/demo/` |
| 200+ unit tests | ✅ `go test ./... -cover` = 84.5% |

### Catatan QR Code

QR WhatsApp di terminal menggunakan Unicode half-block characters.
Jika QR tidak bisa discan dari terminal, buka link URL yang tercetak di bawah QR
di browser HP — WhatsApp dapat scan langsung dari browser.

## Presets

| Preset | Max/min | Max/hour | Max/day | Min delay | Max delay | Cocok untuk |
|--------|---------|----------|---------|-----------|-----------|-------------|
| conservative | 2 | 30 | 150 | 3s | 12s | Akun baru / risiko tinggi |
| moderate | 5 | 60 | 400 | 1.5s | 8s | Penggunaan normal |
| aggressive | 12 | 120 | 800 | 500ms | 4s | Bot aktif |
| high-volume | 30 | 300 | 2000 | 200ms | 2s | Broadcast / high traffic |

## Konfigurasi

Override preset dengan config custom:

```go
cfg := antiban.DefaultConfig(antiban.PresetModerate)
cfg.MaxPerMinute = 8
cfg.WarmUpDays = 14
cfg.EnableTypoInjection = true
cfg.TypoProbability = 0.02
cfg.EnableZeroWidth = true
cfg.EnableEmojiPadding = true
cfg.GroupLurkPeriod = 5 * time.Minute
cfg.MaxStrangerPerDay = 10

abc := antiban.WrapClient(client, antiban.PresetModerate, cfg)
```

Atau gunakan langsung AntiBan tanpa wrapper:

```go
ab := antiban.New(antiban.PresetConservative, cfg)
delay, allowed := ab.BeforeSend(chatID, content)
if allowed {
    time.Sleep(delay)
    // kirim pesan...
    ab.AfterSend(chatID, true)
}
```

## Struktur Proyek

```
whatsmeow-antiban/
├── antiban.go              # Orchestrator utama
├── rate_limiter.go         # Rate limiter dengan Gaussian jitter
├── warmup.go               # Graduated warmup
├── health.go               # Health monitor
├── circuit_breaker.go      # Per-JID circuit breaker
├── timelock_guard.go       # 463 error handler
├── contact_graph.go        # Contact relationship manager
├── content_variator.go     # Variasi konten pesan
├── device_fingerprint.go   # Fingerprint perangkat
├── scheduler.go            # Jadwal pengiriman
├── proxy_rotator.go        # Rotasi proxy
├── ban_recovery.go         # Recovery setelah kena ban
├── reconn_throttle.go      # Throttle setelah reconnect
├── retry_tracker.go        # Tracking alasan retry
├── delivery_tracker.go     # Tracking delivery rate
├── group_guard.go          # Rate limit operasi grup
├── lid_resolver.go         # LID↔PN resolver
├── jid_canonicalizer.go    # JID canonicalization
├── presets.go              # Preset definitions
├── profiles.go             # JID profile helpers
├── persist.go              # State persistence
├── types.go                # Tipe data dan config
├── wrapper.go              # Drop-in wrapper untuk whatsmeow.Client
├── whatsmeow/              # whatsmeow library (embedded)
├── example/
│   ├── main.go             # Contoh dengan koneksi WhatsApp real
│   └── demo/
│       └── main.go         # Demo standalone tanpa WhatsApp
├── *_test.go               # 200+ unit tests (84.5% coverage)
└── README.md
```

## Testing

```bash
go test ./... -v       # verbose
go test ./... -cover   # dengan coverage
```

## Docker Deployment

### One-Line Install (Rekomendasi)

Tanpa perlu copy source code. Cukup curl dan bash:

```bash
curl -sL https://raw.githubusercontent.com/ahlikomputerit/whatpplg/main/install.sh | bash
```

Atau dengan opsi:

```bash
curl -sL https://raw.githubusercontent.com/ahlikomputerit/whatpplg/main/install.sh | \
  bash -s -- --port 8080 --dir /opt/wa-gateway
```

Script akan:
1. Generate `config.yaml` + API key random
2. Pull image dari GitHub Container Registry
3. Jalankan container dengan volume persisten
4. Print API endpoint + key + contoh curl

Hasilnya: **satu container siap pakai**, tanpa source code di PC target.

### Manual (docker run)

```bash
docker pull ghcr.io/ahlikomputerit/whatpplg:latest

# Buat config.yaml dulu (lihat contoh di atas), lalu:
docker run -d \
  --name wa-gateway \
  --restart unless-stopped \
  -p 8080:8080 \
  -v wa-gateway-data:/app/data \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -e TZ=Asia/Jakarta \
  ghcr.io/ahlikomputerit/whatpplg:latest
```

### Docker Compose (dengan Redis/PostgreSQL)

```bash
# Pull image & start
docker compose up -d

# Dengan Redis
docker compose --profile with-redis up -d

# Cek log
docker compose logs -f

# Berhenti
docker compose down
```

### Development (build dari source)

```bash
# Butuh Air (https://github.com/air-verse/air)
docker compose -f docker-compose.dev.yml up
```

### Volume

| Path | Fungsi |
|------|--------|
| `wa-gateway-data` | Session WA + SQLite |
| `config.yaml` | Konfigurasi gateway |
| `templates/` | Template pesan (optional) |

### First Time: Scan QR

```bash
docker logs -f wa-gateway
# QR code akan muncul. Scan dengan WhatsApp > Linked Devices
# Selanjutnya restart container langsung connect.
```

## Perbandingan dengan baileys-antiban

| Aspek | baileys-antiban | whatsmeow-antiban |
|-------|----------------|-------------------|
| Bahasa | TypeScript (Node.js) | Go |
| Target | WhiskeySockets/Baileys | go.mau.fi/whatsmeow |
| Status | v4.7.0 — production | ~v1.0.0 — siap pakai |
| Tests | 38 file | 23 file (200 test) |
| Coverage | ~90% | 84.5% |

## Lisensi

MIT
