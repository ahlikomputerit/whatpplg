# Desain WA Gateway Service

WhatsApp Gateway Service — layanan pengiriman pesan WhatsApp terpusat
dengan anti-ban, queue, dan API fleksibel yang bisa dikonsumsi dari
berbagai aplikasi dan bahasa pemrograman.

---

## 1. Arsitektur

```
                         ┌──────────────────────┐
                         │   Database Eksternal  │
                         │   (app laporan)       │
                         └──────────────────────┘
                                  │
                          (app langsung baca DB)
                                  │
                                  ▼
┌──────────┐  ┌──────────┐  ┌──────────────────────┐
│ SIAKAD   │  │ Absensi  │  │ Bimbingan Konseling   │
│ (PHP)    │  │ (Python) │  │ (JS/Node)             │
└────┬─────┘  └────┬─────┘  └──────────┬───────────┘
     │             │                    │
     │             │                    │
     ▼             ▼                    ▼
┌────────────────────────────────────────────────────┐
│                  WA Gateway Service                 │
│                                                    │
│  ┌─────────┐  ┌────────┐  ┌────────────────────┐  │
│  │ REST    │  │ Webhook│  │ Database Langsung   │  │
│  │ API     │  │ Caller │  │ (optional)          │  │
│  └────┬────┘  └───┬────┘  └─────────┬──────────┘  │
│       │           │                 │              │
│       ▼           ▼                 ▼              │
│  ┌─────────────────────────────────────────────┐   │
│  │            Pipeline Manager                  │   │
│  │  ┌─────────┐ ┌────────┐ ┌───────────────┐   │   │
│  │  │ Adapter │ │ Queue  │ │ Template Engine│   │   │
│  │  │ Router  │ │ Manager│ │ (optional)     │   │   │
│  │  └────┬────┘ └───┬────┘ └───────┬───────┘   │   │
│  └───────┼──────────┼──────────────┼───────────┘   │
│          ▼          ▼              ▼                │
│  ┌─────────────────────────────────────────────┐   │
│  │         AntiBan Pipeline                     │   │
│  │  WarmUp → RateLimit → Scheduler → Circuit   │   │
│  │  → ContactGraph → ContentVar → KIRIM        │   │
│  └──────────────────┬──────────────────────────┘   │
│                     │                               │
│                     ▼                               │
│              WhatsApp Server                        │
└────────────────────────────────────────────────────┘
```

## 2. Mode Integrasi

Gateway mendukung **3 mode** agar bisa menangani berbagai model API dari aplikasi manapun:

### Mode A: REST API (Standar)

Untuk aplikasi yang bisa HTTP POST.

```
POST /api/v1/send
Content-Type: application/json

{
  "to": "62812xxxx",                    // nomor tujuan (tanpa @s.whatsapp.net)
  "message": "Assalamu'alaikum...",     // plain text atau JSON object
  "source": "siapad",                   // identitas pengirim (untuk log)
  "priority": 0,                        // 0=normal, 1=tinggi
  "schedule_at": null,                  // ISO8601 atau null (kirim sekarang)
  "idempotency_key": "abc-123",         // optional, cegah duplikat
  "template": null,                     // optional, nama template
  "template_data": null                 // optional, data untuk template
}
```

Response:

```json
{
  "status": "queued",
  "id": "msg_abc123",
  "estimated_delay_ms": 3200
}
```

### Mode B: Webhook Inbound

Gateway bisa dipanggil balik oleh aplikasi via webhook. Cocok untuk
aplikasi yang tidak bisa HTTP POST (legacy) atau yang events-driven.

Aplikasi daftarkan webhook URL ke gateway, lalu gateway akan GET/POST
ke URL tersebut untuk mengambil daftar pesan yang harus dikirim.

```
Registrasi Webhook:
POST /api/v1/webhook
{
  "name": "laporan-siakad",
  "url": "https://siakad.sch.id/wa-webhook",
  "method": "GET",
  "interval": 60,                    // cek setiap 60 detik
  "headers": {"Authorization": "Bearer xxx"},
  "response_mapping": {              // mapping respons ke format gateway
    "to_field": "data.[].nomor_wa",
    "message_field": "data.[].pesan",
    "id_field": "data.[].id"
  }
}
```

Gateway akan memanggil webhook secara berkala:

```
# Gateway GET ke webhook
GET https://siakad.sch.id/wa-webhook
Response:
{
  "data": [
    {
      "id": 1,
      "nomor_wa": "62812xxxx",
      "pesan": "Assalamu'alaikum..."
    },
    {
      "id": 2,
      "nomor_wa": "62813xxxx",
      "pesan": "Assalamu'alaikum..."
    }
  ]
}

# Setelah terkirim, gateway POST status balik
POST https://siakad.sch.id/wa-webhook/callback
{
  "results": [
    {"id": 1, "status": "sent", "error": null},
    {"id": 2, "status": "failed", "error": "invalid number"}
  ]
}
```

### Mode C: Database Listener (Tanpa API)

Gateway bisa membaca langsung dari database aplikasi lain.
Cocok untuk sistem monolitik atau legacy yang tidak bisa diubah kodenya.

Gateway terkoneksi ke database yang sama, lalu polling tabel tertentu.

```
Config gateway:
{
  "source_name": "laporan-siakad",
  "mode": "database",
  "database": {
    "driver": "mysql",
    "dsn": "user:pass@tcp(localhost:3306)/siakad",
    "query": "SELECT id, nomor_wa, pesan FROM wa_outbox WHERE status = 'pending' LIMIT 10",
    "mark_sent": "UPDATE wa_outbox SET status = 'sent', sent_at = NOW() WHERE id = ?",
    "mark_failed": "UPDATE wa_outbox SET status = 'failed', error = ? WHERE id = ?",
    "interval": 30
  },
  "field_mapping": {
    "to": "nomor_wa",
    "message": "pesan",
    "id": "id"
  }
}
```

Gateway akan:

1. Setiap 30 detik, jalankan `query`
2. Ambil row yang `status = 'pending'`
3. Kirim via WA + anti-ban
4. Update status via `mark_sent` atau `mark_failed`

**Aplikasi cukup INSERT ke tabel `wa_outbox`**, tanpa perlu tahu soal WA.

## 3. API Endpoint Lengkap

### Pesan

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| POST | /api/v1/send | Kirim 1 pesan |
| POST | /api/v1/send-bulk | Kirim banyak pesan (max 100) |
| POST | /api/v1/send-template | Kirim pake template terdaftar |
| GET | /api/v1/messages/{id} | Cek status pesan |
| POST | /api/v1/messages/{id}/cancel | Batalkan pesan yang pending |

### Template

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| POST | /api/v1/templates | Daftarkan template pesan |
| GET | /api/v1/templates | Lihat semua template |
| PUT | /api/v1/templates/{name} | Update template |
| DELETE | /api/v1/templates/{name} | Hapus template |

### Sources / Aplikasi

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| POST | /api/v1/sources | Daftarkan sumber aplikasi |
| GET | /api/v1/sources | Lihat semua sumber |
| PUT | /api/v1/sources/{name} | Update konfigurasi source |
| DELETE | /api/v1/sources/{name} | Hapus source |

### Webhook (Mode B)

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| POST | /api/v1/webhooks | Daftarkan webhook source |
| GET | /api/v1/webhooks | Lihat semua webhook |
| PUT | /api/v1/webhooks/{id} | Update webhook |
| DELETE | /api/v1/webhooks/{id} | Hapus webhook |
| POST | /api/v1/webhooks/{id}/test | Test webhook |

### Control

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| GET | /api/v1/stats | Statistik anti-ban + pengiriman |
| GET | /api/v1/health | Health check service |
| POST | /api/v1/pause | Pause semua pengiriman |
| POST | /api/v1/resume | Resume pengiriman |
| GET | /api/v1/logs?source=...&status=... | Log pengiriman |

## 4. Template Engine

Gateway punya template engine biar aplikasi tidak perlu render pesan.

Daftarkan template:

```
POST /api/v1/templates
{
  "name": "laporan-pelanggaran",
  "body": "Assalamu'alaikum Wr. Wb.\n\nYth. Bpk/Ibu {nama_ortu}\n\nDiberitahukan bahwa {nama_siswa} kelas {kelas} telah melakukan pelanggaran:\n\n{kategori}: {deskripsi}\n\nPoin: {poin}\nSanksi: {sanksi}\n\nTerima kasih.\n\n- {sekolah}"
}
```

Gunakan template saat kirim:

```
POST /api/v1/send-template
{
  "to": "62812xxxx",
  "template": "laporan-pelanggaran",
  "data": {
    "nama_ortu": "Budi",
    "nama_siswa": "Andi",
    "kelas": "X-A",
    "kategori": "Terlambat",
    "deskripsi": "Datang pukul 07:45 tanpa izin",
    "poin": 5,
    "sanksi": "Membersihkan kelas 3 hari",
    "sekolah": "SMA N 1 Jakarta"
  }
}
```

Aplikasi tidak perlu tahu format pesan — cukup kirim data JSON.

## 5. Format API yang Didukung

Gateway otomatis mendeteksi format pesan:

### Plain Text
```json
{"message": "Halo, ini pesan text biasa"}
```

### Template dengan Data
```json
{
  "template": "laporan-pelanggaran",
  "data": {"nama_siswa": "Andi", ...}
}
```

### Rich Media (dari aplikasi)
```json
{
  "message": "Lihat foto berikut:",
  "media": {
    "type": "image",
    "url": "https://storage.sch.id/foto-pelanggaran.jpg",
    "caption": "Foto bukti pelanggaran"
  }
}
```

### Forward dari Database
```json
// Cukup INSERT ke tabel wa_outbox, gateway otomatis proses
{
  "nomor_wa": "62812xxxx",
  "pesan": "Assalamu'alaikum...",
  "status": "pending"
}
```

## 6. Database Gateway

### Tabel `messages`

| Kolom | Tipe | Keterangan |
|-------|------|------------|
| id | UUID | Primary key |
| source | VARCHAR | Identitas aplikasi pengirim |
| to_jid | VARCHAR | Nomor tujuan + @s.whatsapp.net |
| message_text | TEXT | Isi pesan (plain) |
| template_name | VARCHAR | Nama template (jika pakai) |
| template_data | JSON | Data template |
| media_url | TEXT | URL media (optional) |
| status | ENUM | queued/sending/sent/failed/cancelled |
| priority | INT | 0 normal, 1 high |
| schedule_at | TIMESTAMP | Jadwal kirim |
| sent_at | TIMESTAMP | Waktu terkirim |
| error_msg | TEXT | Pesan error |
| retry_count | INT | Jumlah retry |
| idempotency_key | VARCHAR | Cegah duplikat |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

### Tabel `sources`

| Kolom | Tipe | Keterangan |
|-------|------|------------|
| name | VARCHAR | Nama aplikasi (siapad, absensi, dll) |
| mode | ENUM | api / webhook / database |
| config | JSON | Konfigurasi (webhook URL, DB DSN, dll) |
| is_active | BOOLEAN | |
| api_key | VARCHAR | Auth key untuk API |
| created_at | TIMESTAMP | |

### Tabel `templates`

| Kolom | Tipe | Keterangan |
|-------|------|------------|
| name | VARCHAR | Nama template |
| body | TEXT | Body dengan placeholder {var} |
| variables | JSON | Daftar variable yang dibutuhkan |
| created_at | TIMESTAMP | |

### Tabel `logs`

| Kolom | Tipe | Keterangan |
|-------|------|------------|
| id | BIGINT | |
| message_id | UUID | |
| source | VARCHAR | |
| event | VARCHAR | queued/sent/failed/retry |
| detail | TEXT | |
| created_at | TIMESTAMP | |

## 7. Skalabilitas & Anti-Ban

Semua pesan dari berbagai sumber melewati **satu pipeline anti-ban** yang sama:

```
[SIAKAD] \
[Absensi]  --> [QUEUE] --> [AntiBan Pipeline] --> [WhatsApp]
[BK]      /
```

**Keuntungan:**
- Rate limit terpusat (tidak overload per jam)
- Satu session WA (tidak perlu scan QR berulang)
- Queue otomatis (tidak blocking)
- Retry + backoff otomatis
- Log terpusat (semua aplikasi kelihatan)

## 8. Contoh Integrasi

### Dari SIAKAD (PHP/Laravel)

```php
// Cukup INSERT ke tabel wa_outbox di database yang sama
DB::table('wa_outbox')->insert([
    'nomor_wa' => $siswa->nomor_ortu,
    'pesan'    => "Assalamu'alaikum...",
    'status'   => 'pending',
]);
// Gateway otomatis ambil dan kirim
```

Atau via REST:

```php
Http::post('http://wa-gateway:8080/api/v1/send', [
    'to'      => $siswa->nomor_ortu,
    'message' => $pesan,
    'source'  => 'siapad',
]);
```

### Dari Absensi (Python)

```python
requests.post("http://wa-gateway:8080/api/v1/send-template", json={
    "to": siswa.nomor_wa,
    "template": "laporan-pelanggaran",
    "data": {
        "nama_siswa": siswa.nama,
        "kelas": siswa.kelas,
        "kategori": "Tidak absen",
        "deskripsi": "Tidak hadir tanpa keterangan"
    }
})
```

### Dari Legacy App (via Database)

```sql
-- Cukup INSERT, gateway yang proses
INSERT INTO wa_outbox (nomor_wa, pesan, status)
VALUES ('62812xxxx', 'Pesan...', 'pending');
```

## 9. Struktur Folder (Kode)

```
wa-gateway/
├── main.go                    # Entry point
├── config/
│   └── config.go              # Konfigurasi YAML/JSON
├── api/
│   ├── router.go              # HTTP router (chi/gin/echo)
│   ├── handler_send.go        # POST /api/v1/send
│   ├── handler_template.go    # CRUD template
│   ├── handler_source.go      # CRUD source/webhook
│   ├── handler_stats.go       # GET /api/v1/stats
│   └── middleware.go           # Auth, logging, rate limit
├── core/
│   ├── pipeline.go            # Pipeline manager
│   ├── queue.go               # Queue (in-memory / Redis / DB)
│   └── engine.go              # Send engine + retry
├── source/
│   ├── source.go              # Interface Source
│   ├── api_source.go          # API source (REST)
│   ├── webhook_source.go      # Webhook polling
│   └── db_source.go           # Database polling
├── template/
│   ├── engine.go              # Template render
│   └── store.go               # Template DB
├── db/
│   ├── mysql.go               # MySQL driver
│   └── migrations/            # SQL migration
├── whatsmeow-antiban/         # Library anti-ban (embedded)
├── go.mod
└── config.yaml                # Config file
```

## 10. Contoh Config

```yaml
server:
  port: 8080
  api_key: "rahasia123"          # Global API key

whatsapp:
  db_path: "wa_session.db"
  preset: "moderate"

sources:
  - name: "siapad"
    mode: "api"
    api_key: "key-siapad-123"    # API key khusus source ini
    allowed_templates: ["laporan-pelanggaran"]

  - name: "absensi"
    mode: "database"
    database:
      driver: "mysql"
      dsn: "user:pass@tcp(db:3306)/absensi"
      query: "SELECT id, nomor_wa, pesan FROM wa_outbox WHERE status='pending' LIMIT 5"
      mark_sent: "UPDATE wa_outbox SET status='sent' WHERE id=?"
      mark_failed: "UPDATE wa_outbox SET status='failed', error=? WHERE id=?"
    interval: 30

  - name: "bimbingan"
    mode: "webhook"
    webhook:
      url: "https://bk.sch.id/api/wa-queue"
      method: "GET"
      interval: 60
      response_mapping:
        to_field: "data.[].nomor_hp"
        message_field: "data.[].isi_pesan"
        id_field: "data.[].id"

queue:
  type: "memory"                  # memory / redis / database
  max_size: 10000

templates:
  - name: "laporan-pelanggaran"
    body: |
      Assalamu'alaikum Wr. Wb.
      
      Yth. Bpk/Ibu {nama_ortu}
      
      Diberitahukan bahwa {nama_siswa} kelas {kelas}
      telah melakukan pelanggaran: {deskripsi}
      
      Terima kasih.
  - name: "info-absensi"
    body: |
      Yth. Orang tua {nama_siswa}
      
      Putra/putri anda tidak hadir pada {tanggal}.
      Hadir: {hadir}
      Sakit: {sakit}
      Izin: {izin}
      Alpha: {alpha}
```

## 11. Deployment dengan Docker

Gateway sudah siap di-deploy dengan Docker.

### File yang diperlukan

```
wa-gateway/
├── Dockerfile              # Multi-stage Go build
├── docker-compose.yml      # Service + Redis + PostgreSQL (opsional)
├── docker-compose.dev.yml  # Hot-reload development
├── config.yaml             # Konfigurasi gateway
├── Makefile                # Shortcut: make up, make logs, dll
├── templates/              # Template pesan (optional)
└── whatsmeow-antiban/      # Library anti-ban (embedded)
```

### Production

```bash
docker compose build
docker compose up -d
docker compose logs -f
```

### Development (hot reload)

```bash
docker compose -f docker-compose.dev.yml up
```

### Dengan Redis (queue)

```bash
docker compose --profile with-redis up -d
```

### Akses

```
API:    http://localhost:8080/api/v1/...
Health: http://localhost:8080/api/v1/health
Adminer (dev): http://localhost:8081
```

### Volume persistensi

| Volume | Mount di container | Fungsi |
|--------|--------------------|--------|
| `gateway-data` | `/app/data` | Session WA + SQLite |
| `./config.yaml` | `/app/config.yaml` | Konfigurasi |
| `./templates` | `/app/templates` | Template pesan |

> Session WA tetap aman selama volume `gateway-data` tidak dihapus.
> Scan QR hanya sekali — setelah itu restart container langsung connect.

## Ringkasan

| Mode | Cocok untuk | Cara Kirim Pesan |
|------|-------------|------------------|
| **REST API** | Aplikasi modern (Laravel, Django, Express) | POST JSON |
| **Webhook Inbound** | Aplikasi yang tidak bisa keluar network | Gateway polling URL |
| **Database Listener** | Legacy / monolitik | Cukup INSERT ke tabel |

Semua mode ujungnya masuk ke **satu pipeline anti-ban** yang sama —
sehingga rate limit akurat, session WA tidak pecah, dan log terpusat.
