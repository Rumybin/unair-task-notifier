# 📚 Unair Task Notifier

> Scraper otomatis tugas Moodle HEBAT Unair → notifikasi WhatsApp.  
> 100% gratis, jalan di GitHub Actions, tanpa server 24 jam.

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](#)
[![GitHub Actions](https://img.shields.io/badge/GitHub%20Actions-Cron-2088FF?logo=githubactions)](https://github.com/features/actions)

---

## ✨ Fitur

| Fitur | Deskripsi |
|-------|-----------|
| 🔍 **Scraping Otomatis** | Ambil daftar tugas dari dashboard HEBAT Unair tiap jam |
| 🔔 **Notifikasi WhatsApp** | Kirim pesan via Callmebot saat ada tugas baru |
| ⚠️ **Deteksi Perubahan** | Tahu jika deadline tugas berubah |
| 💾 **Snapshot Database** | Simpan riwayat tugas di MariaDB (PlanetScale/Railway) |
| 🕒 **Cron Job** | Jalan otomatis setiap jam via GitHub Actions |
| 🆓 **100% Gratis** | Tidak perlu server — GitHub Actions + free tier DB |

---

## 🏗️ Arsitektur

```
                     ┌─────────────┐
                     │  GitHub      │
                     │  Actions     │
                     │  (Cron/jam)  │
                     └──────┬──────┘
                            │
              ┌─────────────┼─────────────┐
              │             │             │
              ▼             ▼             ▼
       ┌──────────┐  ┌──────────┐  ┌──────────┐
       │ HEBAT    │  │ MariaDB  │  │ Callmebot│
       │ Unair    │  │ (FreeDB) │  │ (WhatsApp)│
       │ (Moodle) │  │          │  │          │
       └──────────┘  └──────────┘  └──────────┘
```

### Alur Eksekusi

```
1. Load environment variables (GitHub Secrets)
2. Connect ke MariaDB → buat tabel tasks jika belum ada
3. Login ke portal HEBAT Unair (Moodle)
4. Scrape halaman /my/ → parse daftar tugas
5. Bandingkan dengan snapshot database
6. Kirim notifikasi WA untuk tugas baru / deadline berubah
7. Simpan snapshot terbaru ke database
```

---

## 🛠️ Tech Stack

| Komponen | Teknologi | Alasan |
|----------|-----------|--------|
| **Bahasa** | [Go](https://go.dev) | Binary kecil, cepat, cocok untuk CI/CD |
| **Scheduler** | [GitHub Actions](https://github.com/features/actions) | Cron gratis, terintegrasi dengan repo |
| **Database** | MariaDB (PlanetScale/Railway free) | Simpan snapshot tugas |
| **Notifikasi** | [Callmebot API](https://www.callmebot.com/) | WhatsApp gratis, tanpa QR/bot |
| **Parser HTML** | `golang.org/x/net/html` | Standard Go extended library |

---

## 📁 Struktur Project

```
unair-task-notifier/
├── .github/workflows/
│   └── scrape.yml              ← GitHub Actions cron (tiap jam)
├── cmd/scraper/
│   └── main.go                 ← Entrypoint / orchestrator
├── internal/
│   ├── auth/login.go           ← Login ke Moodle HEBAT
│   ├── diff/diff.go            ← Bandingkan tugas baru vs lama
│   ├── notifier/callmebot.go   ← Kirim notifikasi WhatsApp
│   ├── scraper/tasks.go        ← Scrape halaman dashboard
│   └── storage/mariadb.go      ← Koneksi & operasi MariaDB
├── testdata/
│   └── sample_task_page.html   ← HTML fixture untuk testing
├── .env.example                ← Template environment variables
├── .gitignore
└── README.md
```

---

## 🚀 Cara Setup (Satu Kali)

### 1. Clone repo

```bash
git clone https://github.com/Rumybin/unair-task-notifier.git
cd unair-task-notifier
```

### 2. Setup Database (MariaDB)

Buat database gratis di [PlanetScale](https://planetscale.com) atau [Railway](https://railway.app), lalu dapatkan **DSN** (`user:pass@tcp(host:port)/dbname`).

### 3. Daftar Callmebot

1. Kunjungi [callmebot.com](https://www.callmebot.com/)
2. Ikuti instruksi aktivasi WhatsApp
3. Dapatkan **API Key** dan **nomor telepon** (format: `628xxxxxxxxx`)

### 4. Set GitHub Secrets

Buka repo → **Settings** → **Secrets and variables** → **Actions** → **New repository secret**:

| Secret | Deskripsi |
|--------|-----------|
| `UNAIR_USERNAME` | NIM (username Moodle HEBAT) |
| `UNAIR_PASSWORD` | Password Moodle HEBAT |
| `MARIADB_DSN` | Connection string MariaDB |
| `CALLMEBOT_PHONE` | Nomor WA (format: 628xx) |
| `CALLMEBOT_APIKEY` | API Key Callmebot |

### 5. Aktifkan Workflow

Di repo GitHub, buka tab **Actions** → **Scrape Unair Tasks** → **Enable**.

Workflow akan jalan otomatis setiap jam. Bisa juga di-trigger manual via **Run workflow**.

---

## 🧪 Testing Lokal

```bash
# Test semua package
go test ./...

# Test dengan coverage
go test -cover ./...

# Build binary
go build ./cmd/scraper -o scraper
```

---

## 📦 Environment Variables (.env.example)

```env
# Moodle HEBAT Unair credentials
UNAIR_USERNAME=
UNAIR_PASSWORD=

# MariaDB connection string
# Format: user:pass@tcp(host:port)/dbname
MARIADB_DSN=

# Callmebot WhatsApp API
CALLMEBOT_PHONE=
CALLMEBOT_APIKEY=
```

> **⚠️ Jangan commit file `.env`** — sudah di-ignore oleh `.gitignore`.

---

## 📊 Database Schema

```sql
CREATE TABLE tasks (
    id          VARCHAR(255) PRIMARY KEY,   -- event-id dari Moodle
    course_name VARCHAR(500) NOT NULL,
    title       VARCHAR(500) NOT NULL,
    due_date    DATETIME,
    task_url    VARCHAR(500),
    detected_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

---

## 📎 Catatan

- **Scraper** menggunakan selector HTML yang sudah diverifikasi dari halaman `/my/` portal HEBAT Unair (Moodle 4.x)
- **Error handling**: Semua error fatal akan membuat GitHub Actions job gagal (exit code 1), yang akan memicu notifikasi email dari GitHub
- **Rate limit**: Callmebot membatasi ~1 pesan per menit — cukup untuk notifikasi tugas baru
- **Keamanan**: Semua credential via GitHub Secrets, tidak ada hardcode di kode

---

## 📄 License

MIT © [Rumybin](https://github.com/Rumybin)

---

<div align="center">
  <b>Dibuat untuk kebutuhan sehari-hari mahasiswa Universitas Airlangga</b>
  <br>
  🎓 UNAIR | HEBAT E-Learning
</div>
