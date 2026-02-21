// Package store provides the task persistence layer.
package store

import (
	"database/sql"
	"fmt"
	"time"
)

// TaskStatus represents the lifecycle state of a task.
type TaskStatus string

const (
	StatusPending TaskStatus = "pending"
	StatusRunning TaskStatus = "running"
	StatusDone    TaskStatus = "done"
	StatusFailed  TaskStatus = "failed"
	StatusDead    TaskStatus = "dead" // exceeded max attempts
)

// Task is the persistent unit of work.
type Task struct {
	ID          string
	Type        string
	Payload     []byte
	Status      TaskStatus
	Schedule    string // cron expression; empty = one-shot
	NextRunAt   time.Time
	Attempts    int
	MaxAttempts int
	LastError   string
	WorkerID    string // set when status=running
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Store is the task persistence abstraction.
type Store interface {
	// Enqueue inserts a new task into the store.
	Enqueue(task *Task) error

	// ClaimNext atomically sets one pending task to running and returns it.
	// Returns (nil, nil) if no tasks are ready.
	ClaimNext(workerID string) (*Task, error)

	// MarkDone marks a task as completed successfully.
	MarkDone(taskID string) error

	// MarkFailed marks a task as failed, stores the error message.
	// If attempts < maxAttempts, status goes back to pending with backoff.
	// If attempts >= maxAttempts, status becomes dead.
	MarkFailed(taskID string, errMsg string) error

	// Reschedule updates next_run_at for a recurring task after completion.
	Reschedule(taskID string, nextRun time.Time) error

	// ListDead returns up to limit tasks in 'dead' status.
	ListDead(limit int) ([]*Task, error)

	// Retry moves a dead task back to pending.
	Retry(taskID string) error

	// Stats returns queue statistics.
	Stats() (QueueStats, error)
}

// QueueStats holds observable queue metrics.
type QueueStats struct {
	Pending int64
	Running int64
	Done    int64
	Failed  int64
	Dead    int64
}

// ─────────────────────────────────────────────────────────────
// SQLite Implementation
// ─────────────────────────────────────────────────────────────

// SQLiteStore is a Store backed by SQLite via database/sql.
// SQLite3 driver: use modernc.org/sqlite (pure Go, no CGO) or mattn/go-sqlite3.
type SQLiteStore struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS tasks (
    id           TEXT PRIMARY KEY,
    type         TEXT NOT NULL,
    payload      BLOB,
    status       TEXT NOT NULL DEFAULT 'pending',
    schedule     TEXT NOT NULL DEFAULT '',
    next_run_at  DATETIME NOT NULL,
    attempts     INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    last_error   TEXT NOT NULL DEFAULT '',
    worker_id    TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL,
    updated_at   DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tasks_status_next ON tasks(status, next_run_at);
`

// Open opens or creates a SQLite database at path and runs migrations.
func Open(path string) (*SQLiteStore, error) {
	// TODO: open database/sql connection
	// TODO: exec schema DDL
	// Return &SQLiteStore{db: db}, nil
	_ = path
	return nil, fmt.Errorf("SQLiteStore.Open: not yet implemented")
}

func (s *SQLiteStore) Enqueue(task *Task) error {
	// TODO: INSERT INTO tasks (...) VALUES (...)
	// Generate UUID for task.ID if empty
	_ = task
	return fmt.Errorf("Enqueue: not yet implemented")
}

func (s *SQLiteStore) ClaimNext(workerID string) (*Task, error) {
	// TODO: In a transaction:
	//   SELECT id FROM tasks WHERE status='pending' AND next_run_at <= ? LIMIT 1
	//   UPDATE tasks SET status='running', worker_id=?, updated_at=? WHERE id=?
	// This "optimistic lock" works because SQLite serializes writers.
	_ = workerID
	return nil, fmt.Errorf("ClaimNext: not yet implemented")
}

func (s *SQLiteStore) MarkDone(taskID string) error {
	// TODO: UPDATE tasks SET status='done', updated_at=now WHERE id=?
	_ = taskID
	return fmt.Errorf("MarkDone: not yet implemented")
}

func (s *SQLiteStore) MarkFailed(taskID string, errMsg string) error {
	// TODO: Increment attempts; if attempts >= max_attempts → status='dead'
	// else → status='pending', next_run_at = now + exponential_backoff(attempts)
	_, _ = taskID, errMsg
	return fmt.Errorf("MarkFailed: not yet implemented")
}

func (s *SQLiteStore) Reschedule(taskID string, nextRun time.Time) error {
	// TODO: UPDATE tasks SET status='pending', next_run_at=? WHERE id=?
	_, _ = taskID, nextRun
	return fmt.Errorf("Reschedule: not yet implemented")
}

func (s *SQLiteStore) ListDead(limit int) ([]*Task, error) {
	// TODO: SELECT * FROM tasks WHERE status='dead' ORDER BY updated_at DESC LIMIT ?
	_ = limit
	return nil, fmt.Errorf("ListDead: not yet implemented")
}

func (s *SQLiteStore) Retry(taskID string) error {
	// TODO: UPDATE tasks SET status='pending', attempts=0, next_run_at=now, updated_at=now WHERE id=? AND status='dead'
	_ = taskID
	return fmt.Errorf("Retry: not yet implemented")
}

func (s *SQLiteStore) Stats() (QueueStats, error) {
	// TODO: SELECT status, COUNT(*) FROM tasks GROUP BY status
	return QueueStats{}, fmt.Errorf("Stats: not yet implemented")
}

// backoffDuration returns exponential backoff with jitter for attempt n.
func backoffDuration(attempt int) time.Duration {
	base := time.Second
	max := 10 * time.Minute
	d := base * (1 << attempt) // 1s, 2s, 4s, 8s...
	if d > max {
		d = max
	}
	// TODO: add up to 500ms jitter using rand.Int63n(500)
	return d
}
