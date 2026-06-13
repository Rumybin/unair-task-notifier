package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/Rumybin/unair-task-notifier/internal/diff"
)

const defaultTimeout = 10 * time.Second

func Connect(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("storage: DSN is empty")
	}
	db, err := sql.Open("mysql", dsn+"?parseTime=true")
	if err != nil {
		return nil, fmt.Errorf("storage: open connection: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("storage: ping failed: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

func EnsureTableExists(db *sql.DB) error {
	query := "CREATE TABLE IF NOT EXISTS tasks (" +
		"id VARCHAR(255) PRIMARY KEY," +
		"course_name VARCHAR(500) NOT NULL," +
		"title VARCHAR(500) NOT NULL," +
		"due_date DATETIME," +
		"task_url VARCHAR(500)," +
		"detected_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP" +
		")"
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("storage: ensure tasks table: %w", err)
	}
	return nil
}

func LoadAllTasks(db *sql.DB) ([]diff.Task, error) {
	query := "SELECT id, course_name, title, due_date, task_url FROM tasks"
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("storage: load all tasks: %w", err)
	}
	defer rows.Close()
	var tasks []diff.Task
	for rows.Next() {
		var t diff.Task
		var dueDate sql.NullTime
		if err := rows.Scan(&t.ID, &t.CourseName, &t.Title, &dueDate, &t.TaskURL); err != nil {
			return nil, fmt.Errorf("storage: scan row: %w", err)
		}
		if dueDate.Valid {
			t.DueDate = dueDate.Time
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: iterate rows: %w", err)
	}
	return tasks, nil
}

func SaveTasks(db *sql.DB, tasks []diff.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	query := "INSERT INTO tasks (id, course_name, title, due_date, task_url) " +
		"VALUES (?, ?, ?, ?, ?) " +
		"ON DUPLICATE KEY UPDATE " +
		"course_name = VALUES(course_name), " +
		"title = VALUES(title), " +
		"due_date = VALUES(due_date), " +
		"task_url = VALUES(task_url)"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("storage: begin tx: %w", err)
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("storage: prepare upsert: %w", err)
	}
	defer stmt.Close()
	for _, t := range tasks {
		var dueDate *time.Time
		if !t.DueDate.IsZero() {
			dueDate = &t.DueDate
		}
		if _, err := stmt.ExecContext(ctx, t.ID, t.CourseName, t.Title, dueDate, t.TaskURL); err != nil {
			return fmt.Errorf("storage: upsert task %s: %w", t.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("storage: commit tx: %w", err)
	}
	return nil
}
