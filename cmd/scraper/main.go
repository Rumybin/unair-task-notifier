package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Rumybin/unair-task-notifier/internal/auth"
	"github.com/Rumybin/unair-task-notifier/internal/diff"
	"github.com/Rumybin/unair-task-notifier/internal/notifier"
	"github.com/Rumybin/unair-task-notifier/internal/scraper"
	"github.com/Rumybin/unair-task-notifier/internal/storage"
)

const (
	baseURL    = "https://hebat.elearning.unair.ac.id"
	appTimeout = 5 * time.Minute
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)

	ctx, cancel := context.WithTimeout(context.Background(), appTimeout)
	defer cancel()

	username := os.Getenv("UNAIR_USERNAME")
	password := os.Getenv("UNAIR_PASSWORD")
	dsn := os.Getenv("MARIADB_DSN")
	phone := os.Getenv("CALLMEBOT_PHONE")
	apikey := os.Getenv("CALLMEBOT_APIKEY")

	if username == "" || password == "" || dsn == "" || phone == "" || apikey == "" {
		log.Fatalf("semua env var wajib diisi")
	}

	log.Println("Menghubungkan ke MariaDB...")
	db, err := storage.Connect(dsn)
	if err != nil {
		log.Fatalf("koneksi DB: %v", err)
	}
	defer db.Close()

	log.Println("Memastikan tabel tasks tersedia...")
	if err := storage.EnsureTableExists(db); err != nil {
		log.Fatalf("buat tabel: %v", err)
	}

	log.Println("Login ke portal HEBAT Unair...")
	jar, err := auth.Login(ctx, baseURL, username, password)
	if err != nil {
		log.Fatalf("login: %v", err)
	}

	client := auth.NewHTTPClient(jar)

	log.Println("Mengambil daftar tugas dari /my/...")
	freshTasks, err := scraper.FetchTasks(ctx, client, baseURL)
	if err != nil {
		log.Fatalf("scrape tugas: %v", err)
	}
	log.Printf("Ditemukan %d tugas dari portal", len(freshTasks))

	log.Println("Memuat snapshot tugas dari database...")
	storedTasks, err := storage.LoadAllTasks(db)
	if err != nil {
		log.Fatalf("load tasks: %v", err)
	}
	log.Printf("Terdapat %d tugas tersimpan", len(storedTasks))

	newTasks := diff.FindNewTasks(freshTasks, storedTasks)
	changedDeadlines := diff.FindChangedDeadlines(freshTasks, storedTasks)

	notifCtx, notifCancel := context.WithTimeout(ctx, 60*time.Second)
	defer notifCancel()

	for _, t := range newTasks {
		dueStr := "Tidak diketahui"
		if !t.DueDate.IsZero() {
			dueStr = t.DueDate.Format("02 Jan 2006 15:04")
		}
		msg := notifier.FormatNewTaskMessage(t.CourseName, t.Title, dueStr, baseURL+t.TaskURL)
		log.Printf("Tugas baru: %s - %s", t.Title, t.CourseName)
		if err := notifier.SendNotification(notifCtx, phone, apikey, msg); err != nil {
			log.Printf("Gagal kirim notif %s: %v", t.ID, err)
		}
	}

	for _, t := range changedDeadlines {
		msg := notifier.FormatDeadlineChangedMessage(t.Title, baseURL+t.TaskURL)
		log.Printf("Deadline berubah: %s", t.Title)
		if err := notifier.SendNotification(notifCtx, phone, apikey, msg); err != nil {
			log.Printf("Gagal kirim notif deadline %s: %v", t.ID, err)
		}
	}

	log.Println("Menyimpan snapshot tugas ke database...")
	if err := storage.SaveTasks(db, freshTasks); err != nil {
		log.Fatalf("simpan tasks: %v", err)
	}

	log.Println("Selesai.")
}

