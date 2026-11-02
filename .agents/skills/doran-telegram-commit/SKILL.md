---
name: doran-telegram-commit
description: Standar format dan aturan pembuatan commit message berbahasa Indonesia di Doran Telegram Server dengan awalan DORAN TELEGRAM SERVER.
---

# Standar Commit Message Doran Telegram Server

Skill ini mengatur format baku pembuatan commit message di repository `telegram-sender` (Doran Telegram Server).

---

## 1. Aturan Dasar
1. **Awalan Wajib**: Baris pertama **WAJIB** diawali dengan `DORAN TELEGRAM SERVER: <ringkasan>`.
2. **Bahasa**: Wajib menggunakan **Bahasa Indonesia** yang jelas, baku, dan lugas.
3. **Tanpa Emoji**: Dilarang menyertakan emoji dalam commit message kecuali ada permintaan khusus dari user.
4. **Tanpa Paragraf Deskripsi Tambahan**: Tidak perlu paragraf pengantar/penjelas umum di antara header dan bullet points. Langsung header diikuti baris kosong lalu bullet points.
5. **Struktur 2 Bagian**:
   - **Header**: `DORAN TELEGRAM SERVER: <ringkasan tujuan perubahan>`
   - **Detail (Bullet points)**: Poin-poin spesifik perbaikan atau file/modul yang dimodifikasi.

---

## 2. Format Baku

### Format Standar (Multi-line)
```text
DORAN TELEGRAM SERVER: <ringkasan judul perubahan>

- <file/modul>: <detail perbaikan 1>
- <file/modul>: <detail perbaikan 2>
- <file/modul>: <detail perbaikan 3>
```

### Contoh Penerapan:
```text
DORAN TELEGRAM SERVER: Penonaktifan auto-sleep floodwait, penambahan rate limiter API, dan pencatatan detail log per device

- internal/telegram/service.go: Menghapus middleware floodwait auto-sleep dan langsung meneruskan error FLOOD_WAIT ke caller API
- internal/handlers/ratelimit.go: Menambahkan middleware in-memory token bucket rate limiter per API key / IP
- internal/models/log.go: Menambahkan kolom IP, Data payload JSON, serta indeks pada CreatedAt
- internal/services/log_service.go: Menambahkan fungsi pembersihan otomatis log lama dengan retensi 31 hari
- resources/js/Pages/Devices/Logs.tsx: Menambahkan halaman riwayat log device dengan filter level, pencarian, dan penampil JSON payload
```

---

## 3. Checklist Sebelum Commit
1. Cek status file: `rtk git status`
2. Pastikan file relevan sudah masuk stage: `rtk git add <file>`
3. Pastikan kode Go lulus kompilasi: `go build ./...`
4. Pastikan TypeScript lulus kompilasi: `npx tsc --noEmit`
5. Jalankan commit dengan format di atas:
   ```bash
   rtk git commit -m "DORAN TELEGRAM SERVER: <ringkasan judul>

- <detail perbaikan 1>
- <detail perbaikan 2>"
   ```
