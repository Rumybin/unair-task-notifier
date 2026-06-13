package diff

import (
	"testing"
	"time"
)

func mustTime(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04", s)
	if err != nil {
		panic("bad test time: " + s + ": " + err.Error())
	}
	return t
}

// ──────────────────────────────────────────────
// FindNewTasks — test cases
// ──────────────────────────────────────────────

func TestFindNewTasks_ReturnsEmpty_WhenFreshIsEmpty(t *testing.T) {
	// Arrange
	stored := []Task{{ID: "1"}, {ID: "2"}}

	// Act
	got := FindNewTasks(nil, stored)

	// Assert
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestFindNewTasks_ReturnsEmpty_WhenNoNewTasks(t *testing.T) {
	// Arrange
	fresh := []Task{{ID: "1"}, {ID: "2"}}
	stored := []Task{{ID: "1"}, {ID: "2"}}

	// Act
	got := FindNewTasks(fresh, stored)

	// Assert
	if len(got) != 0 {
		t.Errorf("expected 0 new tasks, got %d", len(got))
	}
}

func TestFindNewTasks_ReturnsOnlyNewTasks_WhenSomeExist(t *testing.T) {
	// Arrange
	fresh := []Task{
		{ID: "1", Title: "Tugas 1"},
		{ID: "2", Title: "Tugas 2"},
		{ID: "3", Title: "Tugas 3"},
	}
	stored := []Task{
		{ID: "1", Title: "Tugas 1"},
		{ID: "3", Title: "Tugas 3"},
	}

	// Act
	got := FindNewTasks(fresh, stored)

	// Assert
	if len(got) != 1 {
		t.Fatalf("expected 1 new task, got %d", len(got))
	}
	if got[0].ID != "2" {
		t.Errorf("expected task ID 2, got %s", got[0].ID)
	}
}

func TestFindNewTasks_ReturnsAllFresh_WhenStoredIsEmpty(t *testing.T) {
	// Arrange
	fresh := []Task{
		{ID: "1", Title: "Tugas 1"},
		{ID: "2", Title: "Tugas 2"},
	}

	// Act
	got := FindNewTasks(fresh, nil)

	// Assert
	if len(got) != 2 {
		t.Errorf("expected 2 new tasks, got %d", len(got))
	}
}

// ──────────────────────────────────────────────
// FindChangedDeadlines — test cases
// ──────────────────────────────────────────────

func TestFindChangedDeadlines_ReturnsEmpty_WhenFreshIsEmpty(t *testing.T) {
	// Arrange
	stored := []Task{{ID: "1", DueDate: mustTime("2026-06-15 09:00")}}

	// Act
	got := FindChangedDeadlines(nil, stored)

	// Assert
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestFindChangedDeadlines_ReturnsEmpty_WhenStoredIsEmpty(t *testing.T) {
	// Arrange
	fresh := []Task{{ID: "1", DueDate: mustTime("2026-06-15 09:00")}}

	// Act
	got := FindChangedDeadlines(fresh, nil)

	// Assert
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestFindChangedDeadlines_ReturnsEmpty_WhenDeadlinesSame(t *testing.T) {
	// Arrange
	due := mustTime("2026-06-15 09:00")
	fresh := []Task{{ID: "1", DueDate: due}}
	stored := []Task{{ID: "1", DueDate: due}}

	// Act
	got := FindChangedDeadlines(fresh, stored)

	// Assert
	if len(got) != 0 {
		t.Errorf("expected 0 changed deadlines, got %d", len(got))
	}
}

func TestFindChangedDeadlines_ReturnsTask_WhenDeadlineChanged(t *testing.T) {
	// Arrange
	fresh := []Task{{ID: "1", Title: "Tugas 1", DueDate: mustTime("2026-06-16 23:59")}}
	stored := []Task{{ID: "1", Title: "Tugas 1", DueDate: mustTime("2026-06-15 09:00")}}

	// Act
	got := FindChangedDeadlines(fresh, stored)

	// Assert
	if len(got) != 1 {
		t.Fatalf("expected 1 changed deadline, got %d", len(got))
	}
	if got[0].ID != "1" {
		t.Errorf("expected task ID 1, got %s", got[0].ID)
	}
}

func TestFindChangedDeadlines_OnlyReturnsIdsPresentInBoth(t *testing.T) {
	// Arrange
	fresh := []Task{
		{ID: "1", DueDate: mustTime("2026-06-16 23:59")},
		{ID: "2", DueDate: mustTime("2026-06-15 09:00")}, // not in stored
	}
	stored := []Task{
		{ID: "1", DueDate: mustTime("2026-06-15 09:00")},
	}

	// Act
	got := FindChangedDeadlines(fresh, stored)

	// Assert
	if len(got) != 1 {
		t.Fatalf("expected 1 changed deadline, got %d", len(got))
	}
	if got[0].ID != "1" {
		t.Errorf("expected task ID 1, got %s", got[0].ID)
	}
}
