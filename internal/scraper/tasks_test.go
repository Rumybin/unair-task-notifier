package scraper

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseDashboardPage_ReturnsTasksFromFixture(t *testing.T) {
	f, err := os.Open("../../testdata/sample_task_page.html")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	tasks, err := parseDashboardPage(f)
	if err != nil {
		t.Fatalf("parseDashboardPage: %v", err)
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
	expectedURL := "https://hebat.elearning.unair.ac.id/mod/assign/view.php?id=84121"
	if t1.TaskURL != expectedURL {
		t.Errorf("task[0].TaskURL = %q, want %q", t1.TaskURL, expectedURL)
	}
	// Course name tidak ada di calendar event, jadi mungkin kosong
	// (kecuali fixture juga punya recentlyaccesseditems)
	// timestamp 1781456400 -> 2026-06-15 00:00 WIB = 2026-06-14 17:00 UTC
	expectedDue := time.Date(2026, 6, 14, 17, 0, 0, 0, time.UTC)
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
	// timestamp 1781542800 -> 2026-06-16 00:00 WIB = 2026-06-15 17:00 UTC
	expectedDue2 := time.Date(2026, 6, 15, 17, 0, 0, 0, time.UTC)
	if !t2.DueDate.Equal(expectedDue2) {
		t.Errorf("task[1].DueDate = %s, want %s", t2.DueDate.Format(time.RFC3339), expectedDue2.Format(time.RFC3339))
	}
}

func TestParseDashboardPage_ReturnsEmpty_OnEmptyHTML(t *testing.T) {
	tasks, err := parseDashboardPage(strings.NewReader("<html></html>"))
	if err != nil {
		t.Fatalf("parseDashboardPage: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestParseCourseNameFromDetailPage_ReturnsCourseNameFromBreadcrumb(t *testing.T) {
	f, err := os.Open("../../testdata/sample_detail_page.html")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	courseName, err := parseCourseNameFromDetailPage(f)
	if err != nil {
		t.Fatalf("parseCourseNameFromDetailPage: %v", err)
	}

	expected := "2025Genap - FST25605008 - Fungsi dan Proses Bisnis (Praktikum) - S1 - Sistem Informasi - 2025 - I3"
	if courseName != expected {
		t.Errorf("courseName = %q, want %q", courseName, expected)
	}
}

func TestParseCourseNameFromDetailPage_ReturnsError_OnEmptyHTML(t *testing.T) {
	_, err := parseCourseNameFromDetailPage(strings.NewReader("<html></html>"))
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestParseCourseNameFromDetailPage_ReturnsError_WhenNoBreadcrumb(t *testing.T) {
	html := "<html><body><div>no breadcrumb here</div></body></html>"
	_, err := parseCourseNameFromDetailPage(strings.NewReader(html))
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}
