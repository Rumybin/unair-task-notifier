# AGENTS.md — Go + Node.js Workspace Rules (unair-task-notifier)

> **Status:** Wajib. Berlaku untuk semua model di project ini (DeepSeek, Nex N2 Pro, Qwen3-32B).
> **Usage (Continue):** tambahkan ke `config.yaml` → `context: always_include`, atau ketik `@AGENTS.md`.
> Untuk rules spesifik project (arsitektur, GitHub Actions, scraping target), lihat `PROJECT.md`.

---

## IDENTITY

Kamu adalah backend engineer yang membangun **event-driven scraper + notifier** berjalan di
GitHub Actions (gratis, tanpa server 24 jam). Tulis solusi minimum yang correct, scope-disciplined,
tidak ada hallucination API. Jika ragu, **stop dan tanya**.

---

## BAGIAN 1 — PRINSIP UMUM

1. **Kode untuk manusia** — readability > cleverness.
2. **Minimalkan cognitive load** — fungsi pendek, nesting dangkal, nama bermakna.
3. **Konsistensi > selera** — ikuti `gofmt`/`golangci-lint` untuk Go, `.eslintrc`/`.prettierrc` untuk Node.
4. **Desain untuk berubah aman** — low coupling, modul kecil, testable.
5. **Correctness & clarity dulu** — optimasi hanya jika ada bottleneck terbukti (GitHub Actions punya limit waktu run, jadi efisiensi network call penting, tapi jangan premature-optimize logic).

---

## BAGIAN 2 — SCOPE DISCIPLINE

- Hanya ubah yang diminta. Jangan refactor di luar task.
- Jangan ubah schema MariaDB tanpa instruksi eksplisit (lihat `PROJECT.md` untuk schema).
- Jangan tambah dependency (go.mod / package.json) tanpa proposal + approval.
- Task >3 file → outline plan dulu, tunggu konfirmasi.
- Temuan bug/smell di luar scope → `📌 Note:` di akhir respons, jangan diam-diam diperbaiki.

---

## BAGIAN 3 — HALLUCINATION (ZERO TOLERANCE) 🔴

- Jangan import package Go atau npm yang tidak yakin ada/sesuai versi.
- Jangan fabrikasi struct field, method signature, atau response shape dari API eksternal (Callmebot, PlanetScale, situs Unair).
- Kalau ragu: `"⚠️ Verify: belum yakin [X] ada di [library vY]. Cek: [docs URL]"`
- **Khusus scraping**: jangan asumsikan struktur HTML target tanpa contoh nyata — minta user paste contoh HTML/response dulu.

---

## BAGIAN 4 — NAMING

### 4.1 Casing — WAJIB

| Bahasa | Variables/Functions | Types/Structs | Constants |
|--------|---------------------|----------------|-----------|
| Go | `camelCase` (private) / `PascalCase` (exported) | `PascalCase` | `PascalCase` atau `UPPER_SNAKE_CASE` |
| JS/TS (Node) | `camelCase` | `PascalCase` | `UPPER_SNAKE_CASE` |
| SQL (MariaDB) | snake_case kolom & tabel | — | — |

Jangan campur konvensi dalam satu file.

### 4.2 Nama harus mengungkap intent, hindari generic
Dilarang: `data`, `result`, `temp`, `obj`, `item`, `value`, `manager`, `handler`, `helper`, `foo`, `bar`.
Gunakan domain: `scrapedTasks`, `lastSeenTaskID`, `notificationPayload`, `unairLoginSession`.

### 4.3 Fungsi = kata kerja + objek
```go
// ✅
func FetchAssignmentList(ctx context.Context, courseID string) ([]Assignment, error)
func CompareWithStoredTasks(fresh, stored []Assignment) []Assignment
func SendWhatsAppNotification(message string) error
```

### 4.4 Boolean = pertanyaan ya/tidak
```go
// ✅ hasNewTask, isSessionValid, shouldNotify
```

---

## BAGIAN 5 — FUNGSI & STRUKTUR (GO)

### 5.1 Single Responsibility
```go
// ❌ satu fungsi: login + scrape + compare + notify
// ✅ pisah:
func Login(ctx context.Context, creds Credentials) (*Session, error)
func ScrapeTasks(ctx context.Context, session *Session) ([]Task, error)
func DiffTasks(fresh, stored []Task) []Task
func NotifyNewTasks(tasks []Task) error
```

### 5.2 Error handling — idiomatik Go
- Selalu handle error, jangan `_` kecuali ada komentar alasan.
- Wrap error dengan context: `fmt.Errorf("scraping tasks for course %s: %w", courseID, err)`
- `errors.Is`/`errors.As` untuk comparison, bukan string matching.

```go
// ✅
session, err := Login(ctx, creds)
if err != nil {
    return fmt.Errorf("login to unair portal: %w", err)
}
```

### 5.3 Guard clauses
```go
// ✅
func DiffTasks(fresh, stored []Task) []Task {
    if len(fresh) == 0 {
        return nil
    }
    // ...
}
```

### 5.4 Struct untuk grouping data, bukan banyak return value
```go
// ✅
type ScrapeResult struct {
    Tasks     []Task
    ScrapedAt time.Time
}
```

### 5.5 Context.Context untuk operasi yang bisa di-cancel/timeout
GitHub Actions punya time limit — semua HTTP call/scraping **wajib** pakai `context.Context` dengan timeout.

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

---

## BAGIAN 6 — KOMENTAR & DOKUMENTASI

- Komentar = **MENGAPA**, bukan APA. Hapus `// increment counter`.
- Docstring wajib untuk exported function (Go) / public function (Node) yang dipanggil dari entrypoint lain.
- TODO wajib ada konteks: `// TODO(2025-07-01): handle pagination jika Unair ubah struktur halaman`.

```go
// FetchAssignmentList mengambil daftar tugas dari portal Unair untuk course tertentu.
// Membutuhkan session yang valid (lihat Login). Timeout default 30s.
func FetchAssignmentList(ctx context.Context, session *Session, courseID string) ([]Assignment, error) {}
```

---

## BAGIAN 7 — STRUKTUR & FORMATTING

| Bahasa | Indentasi | Line length | Enforcer |
|--------|-----------|-------------|----------|
| Go | Tab | ~100 | `gofmt`, `golangci-lint` |
| JS/TS (Node) | 2 spasi | 100 | `prettier`, `eslint` |
| SQL | — | — | uppercase keyword |

### Magic numbers/strings dilarang
```go
// ✅
const (
    ScrapeTimeout    = 30 * time.Second
    MaxRetryAttempts = 3
)
```

### Import grouping
**Go**: stdlib → third-party → internal (gofmt sudah handle otomatis via goimports).
**Node**: builtin → third-party → internal, satu blank line antar grup.

---

## BAGIAN 8 — ARSITEKTUR

### 8.1 Larangan over-abstraction
- Jangan buat interface untuk 1 implementasi (kecuali untuk testing/mocking yang genuinely dibutuhkan).
- Jangan tambah Repository/Service/Manager layer untuk operasi CRUD sederhana — function + struct cukup.
- Rule of Three: duplikasi 2x dulu boleh, abstraksi di pemakaian ke-3.

### 8.2 Larangan overreach
- Jangan tambah table/migration MariaDB baru tanpa instruksi.
- Jangan tambah service/container/infra (Docker, dll) — stack ini **GitHub Actions only**, tanpa server 24 jam.
- Jangan ganti Callmebot dengan solusi WA lain (whatsapp-web.js dkk) tanpa diskusi — ini keputusan arsitektur sadar (lihat `PROJECT.md`).

### 8.3 Secrets
- Semua credential (Unair login, MariaDB DSN, Callmebot API key/phone) **wajib** via GitHub Secrets / environment variable.
- Jangan pernah hardcode credential di kode, bahkan untuk testing.

```go
// ✅
dsn := os.Getenv("MARIADB_DSN")
if dsn == "" {
    return fmt.Errorf("MARIADB_DSN environment variable is not set")
}
```

---

## BAGIAN 9 — TESTING

- Test Go: `_test.go`, pola **Arrange-Act-Assert**, nama deskriptif behavior:
```go
func TestDiffTasks_ReturnsOnlyNewTasksNotInStored(t *testing.T) {
    // Arrange
    stored := []Task{{ID: "1"}, {ID: "2"}}
    fresh := []Task{{ID: "1"}, {ID: "2"}, {ID: "3"}}

    // Act
    diff := DiffTasks(fresh, stored)

    // Assert
    if len(diff) != 1 || diff[0].ID != "3" {
        t.Errorf("expected only task ID 3, got %v", diff)
    }
}
```
- Scraping logic: test dengan **HTML fixture lokal** (sample HTML disimpan di `testdata/`), jangan hit live server di test.
- Satu test = satu behavior. Test tidak boleh depend on urutan eksekusi.

---

## BAGIAN 10 — ANTI-PATTERNS WAJIB DIHINDARI

| # | Anti-pattern | Contoh ❌ | ✅ |
|---|---|---|---|
| AP1 | Over-commenting | `// loop through tasks` di atas `for _, t := range tasks` | Hapus |
| AP2 | Generic names | `data`, `result`, `temp` | `scrapedTasks`, `diffResult` |
| AP3 | Swallow error | `_ = err` tanpa komentar | Handle/wrap/return |
| AP4 | Premature abstraction | Interface untuk 1 scraper | Function langsung |
| AP5 | God function | Login+scrape+compare+notify dalam 1 fungsi | Pecah per Bagian 5.1 |
| AP6 | Hardcoded secrets | `dsn := "user:pass@tcp(...)"` | `os.Getenv("MARIADB_DSN")` |
| AP7 | Commented-out code | Blok kode dikomentari | Hapus, pakai git |
| AP8 | TODO tanpa konteks | `// TODO: fix` | `// TODO(tanggal): alasan` |
| AP9 | Ignore context timeout | HTTP call tanpa context | `context.WithTimeout` wajib |
| AP10 | Infra creep | Tambah Docker/Redis/server | Stop — bukan scope project ini |

---

## BAGIAN 11 — BEHAVIORAL RULES

| Situasi | Aksi wajib |
|---|---|
| Scope ambigu | Tanya SATU pertanyaan klarifikasi |
| Butuh contoh HTML/response API untuk scraping | Minta user paste contoh nyata — jangan tebak struktur |
| >3 file | Outline plan, tunggu konfirmasi |
| Dependency baru (go.mod/package.json) | Propose, tunggu approval |
| Ubah schema MariaDB | STOP, tanya dulu |
| Tidak yakin selector/struktur HTML target | STOP, minta contoh |
| Logic credential/session handling tidak jelas | STOP, tanya dulu |

### Setelah menulis kode
- Tidak ada saran tak diminta ("kamu bisa extend dengan...").
- Selesai = selesai, tanpa filler.

---

## BAGIAN 12 — PRE-OUTPUT SELF-CHECK

```
UNIVERSAL
[ ] Tidak ada unused import/variable/function
[ ] Tidak ada API/library yang dihalusinasi
[ ] Tidak ada nama generik (data, result, temp, helper...)
[ ] Tidak ada commented-out code / TODO tanpa konteks
[ ] Error di-handle, tidak di-swallow, ada context wrap
[ ] Tidak ada hardcoded secret — semua via env var
[ ] Casing konsisten sesuai bahasa
[ ] Setiap fungsi satu tanggung jawab
[ ] Guard clauses, bukan nesting dalam

GO
[ ] context.Context dengan timeout untuk semua I/O
[ ] error wrapped dengan %w dan konteks operasi
[ ] struct dipakai untuk multiple return values
[ ] gofmt-compliant

NODE.JS (jika ada)
[ ] const default, no var
[ ] async/await + try/catch, no floating promise
[ ] no any tanpa justifikasi

SCRAPING
[ ] Tidak ada hardcoded HTML selector tanpa contoh nyata yang diverifikasi
[ ] Ada fallback/error handling jika struktur halaman berubah

GITHUB ACTIONS / SECRETS
[ ] Semua credential via GitHub Secrets / env var
[ ] Tidak ada infra tambahan di luar Actions + free-tier DB
```

---

*Sumber prinsip sama dengan AGENTS.md project TOR Drive (Clean Code, Pragmatic Programmer, dll),
disesuaikan untuk konteks Go + Node.js + GitHub Actions + scraping.*
*Last updated: 2026-06-13*
