# PROJECT.md — unair-task-notifier

> Rules & konteks **spesifik project ini**. Untuk rules clean code umum, lihat `AGENTS.md` (auto-loaded, jangan diulang di sini).

---

## TUJUAN PROJECT

Notifier otomatis: scrape daftar tugas dari portal akademik Unair → bandingkan dengan data
tersimpan → jika ada tugas baru, kirim notifikasi WhatsApp ke nomor sendiri.

**Constraint utama:** 100% gratis, online 24/7 tanpa server yang harus selalu nyala.

---

## ARSITEKTUR

```
GitHub Actions (cron, tiap 1 jam)
        │
        ▼
Go scraper — login ke portal Unair, scrape daftar tugas
        │
        ▼
Bandingkan dengan data di MariaDB (PlanetScale free tier)
        │
        ▼
Ada tugas baru? ──Ya──▶ HTTP GET ke Callmebot API
        │
       Tidak
        │
        ▼
Selesai, exit (tidak ada cost, tidak ada server idle)
```

### Komponen

| Komponen | Teknologi | Peran |
|---|---|---|
| Scraper + diff logic | Go | Login, scrape, compare, trigger notif |
| Scheduler | GitHub Actions Cron | Jalankan scraper tiap interval tertentu |
| Database | MariaDB (PlanetScale/Railway free) | Simpan daftar tugas terakhir yang diketahui |
| Notifikasi | Callmebot API | Kirim WA via HTTP GET, tanpa bot/QR |
| (Opsional) Node.js | Coordinator/REST API | Hanya jika dibutuhkan endpoint tambahan — **belum dikonfirmasi perlu atau tidak** |

**Catatan penting:** Posisi Node.js dalam arsitektur ini belum jelas — perlu diklarifikasi
sebelum implementasi: apakah Node.js dipakai untuk endpoint manual trigger, dashboard, atau
tidak dipakai sama sekali (Go saja sudah cukup untuk scraper+diff+notif).

---

## KENAPA CALLMEBOT, BUKAN whatsapp-web.js

- `whatsapp-web.js` butuh sesi QR persistent → butuh storage permanen → tidak cocok untuk
  free-tier yang storage-nya ephemeral/reset.
- Callmebot: HTTP GET sederhana, tidak perlu server nyala, tidak perlu re-scan QR.
- **Keputusan ini sudah final** — jangan disarankan ganti ke solusi WA bot lain kecuali ada
  diskusi ulang eksplisit.

---

## STRUKTUR FOLDER (RENCANA AWAL)

```
unair-task-notifier/
├── AGENTS.md                  # rules clean code Go+Node.js
├── PROJECT.md                 # file ini
├── README.md                  # deskripsi project, cara run
├── .github/
│   └── workflows/
│       └── scrape.yml         # GitHub Actions cron workflow
├── cmd/
│   └── scraper/
│       └── main.go            # entrypoint
├── internal/
│   ├── auth/                  # login ke portal Unair
│   │   └── login.go
│   ├── scraper/                # scraping logic
│   │   └── tasks.go
│   ├── storage/                 # koneksi & query MariaDB
│   │   └── mariadb.go
│   ├── notifier/                 # integrasi Callmebot
│   │   └── callmebot.go
│   └── diff/                     # compare fresh vs stored tasks
│       └── diff.go
├── testdata/                    # HTML fixture untuk testing scraper
│   └── sample_task_page.html
├── go.mod
├── go.sum
└── .env.example                  # template env var (TANPA value asli)
```

> Struktur ini rencana awal — boleh disesuaikan saat implementasi, tapi perubahan struktur
> >3 file harus di-outline dulu (lihat AGENTS.md Bagian 2).

---

## DATA YANG DIBUTUHKAN UNTUK MULAI (BELUM ADA, PERLU DIISI USER)

Sebelum AI bisa mulai coding scraper, **wajib** ada:

1. **URL portal Unair** yang akan di-scrape (contoh: SIAKAD, e-learning, dll — sebutkan platform spesifik)
2. **Contoh HTML/struktur halaman** daftar tugas (paste HTML mentah atau screenshot + inspect element)
3. **Metode login**: form login biasa? SSO? Ada captcha?
4. **Callmebot setup**: nomor WA + API key (dari callmebot.com, gratis, butuh aktivasi via WA)
5. **MariaDB connection**: sudah punya akun PlanetScale/Railway atau belum?

AI **tidak boleh** menebak struktur HTML atau flow login — sesuai AGENTS.md Bagian 3 & 11.

---

## SCHEMA MARIADB (DRAFT — KONFIRMASI DULU)

```sql
CREATE TABLE tasks (
    id VARCHAR(255) PRIMARY KEY,        -- ID unik tugas dari portal Unair
    course_name VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL,
    due_date DATETIME,
    detected_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

> Ini draft — jangan dieksekusi sampai dikonfirmasi cocok dengan data nyata dari portal Unair.

---

## ENVIRONMENT VARIABLES (GitHub Secrets)

```
UNAIR_USERNAME=
UNAIR_PASSWORD=
MARIADB_DSN=
CALLMEBOT_PHONE=
CALLMEBOT_APIKEY=
```

Semua diakses via `os.Getenv(...)` di Go — **tidak ada nilai asli di file manapun di repo**.

---

## GITHUB ACTIONS WORKFLOW (RENCANA)

```yaml
# .github/workflows/scrape.yml (draft)
name: Scrape Unair Tasks
on:
  schedule:
    - cron: "0 * * * *"   # tiap jam, sesuaikan kebutuhan
  workflow_dispatch:        # trigger manual untuk testing

jobs:
  scrape:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go run ./cmd/scraper
        env:
          UNAIR_USERNAME: ${{ secrets.UNAIR_USERNAME }}
          UNAIR_PASSWORD: ${{ secrets.UNAIR_PASSWORD }}
          MARIADB_DSN: ${{ secrets.MARIADB_DSN }}
          CALLMEBOT_PHONE: ${{ secrets.CALLMEBOT_PHONE }}
          CALLMEBOT_APIKEY: ${{ secrets.CALLMEBOT_APIKEY }}
```

> Draft — sesuaikan cron interval dan go-version saat implementasi nyata.

---

## URUTAN IMPLEMENTASI YANG DISARANKAN

1. Setup `go.mod`, struktur folder dasar
2. `internal/storage` — koneksi MariaDB + create table (setelah schema dikonfirmasi)
3. `internal/auth` — login ke portal Unair (setelah flow login dikonfirmasi)
4. `internal/scraper` — scrape daftar tugas (setelah contoh HTML diberikan)
5. `internal/diff` — compare fresh vs stored (paling mudah, pure logic, bisa ditest duluan)
6. `internal/notifier` — integrasi Callmebot (setelah API key didapat)
7. `cmd/scraper/main.go` — orchestrate semua di atas
8. `.github/workflows/scrape.yml` — wire up cron

**Saran:** mulai dari #5 (`diff`) karena pure logic, tidak butuh data eksternal, bisa langsung
ditest. Sambil itu, kumpulkan data #1-4 (contoh HTML, kredensial, dll).

---

## COMMIT FORMAT

```
<type>(scope): <subject under 72 chars>
[optional body — alasan WHY]
```
Types: `feat`, `fix`, `refactor`, `docs`, `chore`. No emoji, lowercase subject.

Contoh: `feat(scraper): add task list parsing for SIAKAD page`

---

*File ini melengkapi AGENTS.md — jangan duplikasi rules umum di sini.*
