# Sistem Laporan Pelanggaran Siswa via WhatsApp

## Gambaran Umum

Sistem otomatis mengirim laporan pelanggaran siswa ke nomor WhatsApp orang tua/wali.
Menggunakan `whatsmeow-antiban` sebagai middleware pengiriman agar akun WhatsApp tidak diblokir.

## Alur Sistem

```
                         +-----------+
                         | Database  |
                         | Siswa     |
                         +-----------+
                              |
                              v
+----------+     +---------------------+     +-------------------+
| Admin    | --> | Scheduler / Trigger | --> | Report Generator  |
| Input    |     | (manual / cron)     |     | (template + data) |
| Laporan  |     +---------------------+     +-------------------+
+----------+                                         |
                                                      v
                              +---------------------+---------------------+
                              |                                           |
                              v                                           v
                      +---------------+                       +-------------------+
                      | AntiBan       |                       | Send Result       |
                      | Pipeline:     |                   --> | Sukses / Gagal    |
                      | - WarmUp      |                  /    +-------------------+
                      | - RateLimit   | <-- whatsmeow-antiban
                      | - ContactGraph|                  \
                      | - Scheduler   |                       +-------------------+
                      | - ContentVar  |                       | Log Pengiriman    |
                      | - CircuitBrkr |                       +-------------------+
                      +---------------+

```

## Komponen Database

### Tabel `siswa`

| Kolom | Tipe | Keterangan |
|-------|------|------------|
| id | INT PK | |
| nis | VARCHAR | Nomor induk siswa |
| nama | VARCHAR | Nama siswa |
| kelas | VARCHAR | |
| nama_ortu | VARCHAR | Nama orang tua |
| nomor_wa | VARCHAR | **Nomor WA orang tua** |
| aktif | BOOLEAN | |

### Tabel `pelanggaran`

| Kolom | Tipe | Keterangan |
|-------|------|------------|
| id | INT PK | |
| siswa_id | INT FK | |
| tgl | DATE | Tanggal kejadian |
| kategori | VARCHAR | Terlambat / Bolos / Berkelahi / dll |
| deskripsi | TEXT | Detail kejadian |
| poin | INT | Bobot pelanggaran |
| sanksi | TEXT | Tindakan sekolah |

### Tabel `laporan_terkirim`

| Kolom | Tipe | Keterangan |
|-------|------|------------|
| id | INT PK | |
| pelanggaran_id | INT FK | |
| nomor_tujuan | VARCHAR | |
| isi_pesan | TEXT | |
| status | ENUM | pending/sukses/gagal |
| error_msg | TEXT | |
| dikirim_at | DATETIME | |

## Template Pesan

```
Assalamu'alaikum Wr. Wb.

Yth. Bapak/Ibu {nama_ortu}

Diberitahukan bahwa putra/putri Bapak/Ibu:
  Nama : {nama_siswa}
  Kelas: {kelas}

Telah melakukan pelanggaran pada {tgl}:
  {kategori}
  {deskripsi}

Poin pelanggaran: {poin}
Sanksi: {sanksi}

Mohon kerjasama Bapak/Ibu untuk membimbing putra/putri
agar tidak mengulangi perbuatan tersebut.

Wassalamu'alaikum Wr. Wb.

- {nama_sekolah}
```

## Cara Kerja Pengiriman

### 1. Admin Input Laporan

Admin mencatat pelanggaran via form (web/desktop). Tersimpan di tabel `pelanggaran`.

### 2. Trigger Pengiriman

Dua mode:

**Mode Manual:**  
Admin klik "Kirim Laporan" → langsung proses antrian.

**Mode Terjadwal (Cron):**  
Setiap jam 19:00 (setelah sekolah), sistem ambil semua pelanggaran hari ini yang belum terkirim → proses antrian.

### 3. Pipeline Anti-Ban

Setiap pesan melewati pipeline `whatsmeow-antiban`:

```
1. Scheduler    → Cek jam aktif (08:00-22:00)
2. WarmUp       → Cek limit harian progresif
3. RateLimiter  → Delay Gaussian jitter, batas per menit/jam/hari
4. ContactGraph → Cek status kontak (stranger → known)
5. CircuitBreak → Cek riwayat gagal nomor tujuan
6. ContentVar   → Variasi teks (biar tidak identical)
7. Kirim        → SendMessage via whatsmeow
8. AfterSend    → Catat statistik, simpan log
```

### 4. Rate Limiting (Preset Moderate)

| Limit | Value | Untuk 100 laporan |
|-------|-------|-------------------|
| Max/menit | 5 | Butuh 20 menit |
| Max/jam | 60 | Butuh ~2 jam |
| Max/hari | 400 | Aman untuk 100 |
| Delay antar pesan | 1.5-8s | Rata-rata ~3 detik |

Untuk **100 laporan**: estimasi selesai dalam **20-30 menit** (tergantung delay).

### 5. Penanganan Gagal

```
Kirim Pesan
    ├── Sukses → Update status = sukses
    │
    └── Gagal
        ├── 405 Forbidden   → CircuitBreaker catat failure
        ├── 463 Timelock    → TimelockGuard blokir sementara
        ├── Timeout         → Retry 3x
        └── Error lain      → Catat di log, manual review
```

## Struktur Kode (Contoh)

```
laporan-sekolah/
├── main.go                 # Entry point + event handler WA
├── db/
│   ├── koneksi.go          # Koneksi database SQLite/MySQL
│   ├── siswa.go            # CRUD data siswa
│   └── laporan.go          # CRUD laporan + tracking kirim
├── pengirim/
│   ├── engine.go           # Antrian pengiriman + pipeline anti-ban
│   └── template.go         # Render template pesan
├── scheduler/
│   └── cron.go             # Cron job pengiriman otomatis
├── web/
│   ├── handler.go          # HTTP handler untuk admin
│   └── template/           # HTML template admin
├── whatsmeow-antiban/      # Library anti-ban (copy)
└── go.mod
```

## Alur Kode Pengiriman

```go
func KirimLaporan(abc *antiban.AntiBanClient, laporan Laporan) {
    // 1. Ambil data siswa
    siswa := db.GetSiswa(laporan.SiswaID)

    // 2. Render template
    pesan := template.Render(laporan, siswa)

    // 3. Parse nomor WA
    jid, _ := types.ParseJID(siswa.NomorWA + "@s.whatsapp.net")

    // 4. Kirim via anti-ban pipeline
    resp, err := abc.SendMessage(ctx, jid, &waE2E.Message{
        Conversation: proto.String(pesan),
    })

    // 5. Catat hasil
    if err != nil {
        db.SimpanLog(laporan.ID, siswa.NomorWA, pesan, "gagal", err.Error())
    } else {
        db.SimpanLog(laporan.ID, siswa.NomorWA, pesan, "sukses", "")
    }
}
```

## Keamanan & Best Practices

1. **WarmUp dulu**: Jangan kirim 100 pesan di hari pertama. Mulai dari 20/hari, naik gradual.
2. **Jeda antar pesan**: Delay 3-8 detik (otomatis oleh anti-ban).
3. **Variasi teks**: Setiap pesan dikasih variasi kecil (spasi, tanda baca) biar tidak detected as spam.
4. **Jam kirim**: Hanya kirim jam 08:00-20:00 (via Scheduler).
5. **Monitoring**: Dashboard status pengiriman, lihat yang gagal.
6. **Backup**: Kalau WA kena ban, masih ada data di database.

## Kapasitas

| Jumlah Laporan/hari | Waktu Kirim | Preset |
|---------------------|-------------|--------|
| 10-30 | 5-15 menit | conservative |
| 30-100 | 20-60 menit | moderate |
| 100-300 | 1-3 jam | aggressive |

> ✅ **Rekomendasi**: Untuk laporan pelanggaran (biasanya 10-50/hari), preset **moderate** sudah cukup aman.

## Kesimpulan

Sistem ini **sangat bisa** dibuat dengan `whatsmeow-antiban`. Library sudah menyediakan semua komponen anti-ban yang diperlukan. Tinggal buat:
1. Database siswa + laporan
2. Template pesan
3. Cron/trigger pengiriman
4. Integrasi dengan `abc.SendMessage()`

Tanpa anti-ban, resiko WA kena spam detection tinggi karena kirim ke banyak nomor baru sekaligus.
