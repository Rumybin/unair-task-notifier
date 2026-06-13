// Package diff menyediakan fungsi untuk membandingkan daftar tugas
// antara hasil scraping terbaru dengan snapshot yang tersimpan di database.
package diff

import "time"

// Task mewakili satu tugas dari portal HEBAT Unair.
type Task struct {
	ID         string    // event-id dari Moodle
	Title      string    // nama tugas
	CourseName string    // nama mata kuliah
	DueDate    time.Time // deadline (jika diketahui)
	TaskURL    string    // URL lengkap ke halaman tugas
}

// FindNewTasks mengembalikan tugas-tugas yang ada di fresh tetapi tidak ada di stored,
// berdasarkan perbandingan ID.
// Urutan hasil tidak dijamin.
func FindNewTasks(fresh, stored []Task) []Task {
	if len(fresh) == 0 {
		return nil
	}

	storedSet := make(map[string]struct{}, len(stored))
	for _, t := range stored {
		storedSet[t.ID] = struct{}{}
	}

	var newTasks []Task
	for _, t := range fresh {
		if _, exists := storedSet[t.ID]; !exists {
			newTasks = append(newTasks, t)
		}
	}

	return newTasks
}

// FindChangedDeadlines mengembalikan tugas-tugas dari fresh yang ID-nya
// sama dengan stored tetapi DueDate-nya berbeda.
// Hanya tugas dari fresh yang dikembalikan (dengan nilai deadline terbaru).
func FindChangedDeadlines(fresh, stored []Task) []Task {
	if len(fresh) == 0 || len(stored) == 0 {
		return nil
	}

	storedMap := make(map[string]time.Time, len(stored))
	for _, t := range stored {
		storedMap[t.ID] = t.DueDate
	}

	var changedTasks []Task
	for _, t := range fresh {
		if storedDue, exists := storedMap[t.ID]; exists {
			if !t.DueDate.Equal(storedDue) {
				changedTasks = append(changedTasks, t)
			}
		}
	}

	return changedTasks
}
