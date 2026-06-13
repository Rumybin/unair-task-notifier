package scraper

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseTasksPage_ReturnsTasksFromFixture(t *testing.T) {
	f, err := os.Open("../../testdata/sample_task_page.html")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	tasks, err := parseTasksPage(f)
	if err != nil {
		t.Fatalf("parseTasksPage: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	t1 := tasks[0]
	if t1.ID != "56362" {
		t.Errorf("task[0].ID = %q, want %q", t1.ID, "56362")
	}
	if t1.Title != "Tugas Minggu 13" {
		t.Errorf("task[0].Title = %q, want %q", t1.Title, "Tugas Minggu 13")
	}
	if t1.TaskURL != "/mod/assign/view.php?id=84121" {
		t.Errorf("task[0].TaskURL = %q", t1.TaskURL)
	}
	courseExpected := "2025Genap - FST25605008 - Fungsi dan Proses Bisnis (Praktikum) - S1 - Sistem Informasi - 2025 - I3"
	if t1.CourseName != courseExpected {
		t.Errorf("task[0].CourseName = %q", t1.CourseName)
	}
	// timestamp 1781456400 -> time.Unix.UTC = 2026-06-14 + jam 09:00
	expectedDue := time.Date(2026, 6, 14, 9, 0, 0, 0, time.UTC)
	if !t1.DueDate.Equal(expectedDue) {
		t.Errorf("task[0].DueDate = %s, want %s", t1.DueDate.Format(time.RFC3339), expectedDue.Format(time.RFC3339))
	}

	t2 := tasks[1]
	if t2.ID != "56370" {
		t.Errorf("task[1].ID = %q, want %q", t2.ID, "56370")
	}
	if t2.Title != "Tugas Minggu 14" {
		t.Errorf("task[1].Title = %q", t2.Title)
	}
	// timestamp 1781542800 -> time.Unix.UTC = 2026-06-15 + jam 23:59
	expectedDue2 := time.Date(2026, 6, 15, 23, 59, 0, 0, time.UTC)
	if !t2.DueDate.Equal(expectedDue2) {
		t.Errorf("task[1].DueDate = %s, want %s", t2.DueDate.Format(time.RFC3339), expectedDue2.Format(time.RFC3339))
	}
}

func TestParseTasksPage_ReturnsEmpty_OnEmptyHTML(t *testing.T) {
	tasks, err := parseTasksPage(strings.NewReader("<html></html>"))
	if err != nil {
		t.Fatalf("parseTasksPage: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

