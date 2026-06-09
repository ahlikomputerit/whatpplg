# Panduan Penggunaan whatsmeow-antiban

## 1. Copy Project ke PC Kamu

Copy folder `whatsmeow-antiban/` ke PC kamu (USB / cloud / git).

## 2. Build

```bash
cd whatsmeow-antiban
go build ./...
```

## 3. Test (Tanpa WhatsApp)

Jalankan demo untuk memastikan semua berfungsi:

```bash
go run ./example/demo/
```

## 4. Jalankan dengan WhatsApp

```bash
go run ./example/ -db whatsmeow.db -preset moderate
```

**Saat pertama kali jalan**, QR Code akan tampil **langsung di terminal** sebagai ASCII art:

```
=== Scan QR Code dengan WhatsApp Anda ===
Buka WhatsApp > Settings > Linked Devices > Link a Device

                ████████████████████████████████
                ██                          ██
                ██  ██████████████████████  ██
                ██  ██                  ██  ██
                ██  ██  ██████████████  ██  ██
                ██  ██  ██          ██  ██  ██
                ██  ██  ██  ████  ██  ██  ██
                ...
```

**Cara pairing:**
1. Buka WhatsApp di HP
2. **Settings (gear icon) > Linked Devices > Link a Device**
3. Scan QR code yang tampil di terminal

⏳ Tunggu sampai muncul **"✅ Terhubung!"**

## 5. Test Fitur

Setelah terhubung, kirim pesan dari HP ke nomor sendiri:

| Perintah | Fungsi |
|----------|--------|
| `!ping` | Bot akan balas "Pong!" — test kirim pesan |
| `!stats` | Bot tampilkan statistik anti-ban (rate limiter, health, dll) |

## 6. Sesi Tersimpan

Sesi login otomatis tersimpan di `whatsmeow.db`. Kedua kalinya jalan, QR code **tidak perlu discan lagi** — langsung connect.

## 7. Hentikan

Tekan `Ctrl+C` untuk berhenti.

## Preset yang Tersedia

| Flag | Kecepatan | Safety |
|------|-----------|--------|
| `-preset conservative` | 2 pesan/menit | Paling aman |
| `-preset moderate` | 5 pesan/menit | Recommended |
| `-preset aggressive` | 12 pesan/menit | Agresif |
| `-preset high-volume` | 30 pesan/menit | Resiko tinggi |

## Troubleshooting

**QR code tidak bisa discan dari terminal?**  
QR WhatsApp menggunakan Unicode half-block characters. Kalau terminal tidak
mendukung, scan dari browser HP dengan link URL yang tercetak di bawah QR.

**Error "sql: unknown driver"?**  
Pastikan Go 1.25+ dan jalankan `go mod tidy` dulu.

**Error "Failed to open database"?**  
Hapus file `whatsmeow.db` lama lalu coba lagi.

**Login ulang tanpa QR?**  
Selama file `.db` masih ada, sesi tersimpan otomatis — tinggal jalankan ulang.

## Referensi

- Source code: `antiban.go` (orchestrator utama)
- Demo: `example/demo/main.go` (tanpa WhatsApp)
- Example: `example/main.go` (dengan WhatsApp)
- Tests: 23 file, 200+ test functions, 84.5% coverage
- Docs: `go doc -all ./...`
