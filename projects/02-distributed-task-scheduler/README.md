# Project 02 — Distributed Task Scheduler

> **Difficulty**: Senior · **Domain**: Distributed Systems, Persistence, CLI
> **Real-world analog**: Temporal, Sidekiq, Celery, AWS Step Functions

---

## Why This Project Exists

Every backend system eventually needs scheduled work: sending emails, generating reports,
retrying failed payments. Most teams use cron + a database as a poor substitute. Production
systems need **at-least-once execution**, **backpressure**, **dead-letter queues**, and
**distributed worker coordination** — this project builds exactly that.

---

## Folder Structure

```
02-distributed-task-scheduler/
├── go.mod
├── main.go                        # CLI: scheduler server + worker + submit
├── scheduler/
│   ├── store/
│   │   ├── store.go               # Storage interface
│   │   └── sqlite.go              # SQLite-backed store via database/sql
│   ├── queue/
│   │   ├── queue.go               # In-process priority queue
│   │   └── dlq.go                 # Dead-letter queue
│   ├── worker/
│   │   ├── pool.go                # Worker pool with backpressure
│   │   └── handler.go             # Task handler registry
│   ├── cron/
│   │   └── parser.go              # Cron expression parser (no external dep)
│   └── server/
│       └── http.go                # REST API: submit, status, retry, cancel
└── testdata/
    └── tasks.json
```

---

## Implementation Guide

### Phase 1 — Cron Expression Parser (Week 1)

Build a cron expression parser that supports the 5-field standard plus `@hourly`, `@daily`,
`@weekly`, `@monthly` shorthands.

```go
sched, _ := cron.Parse("*/5 * * * *")   // every 5 minutes
next := sched.Next(time.Now())           // next fire time
```

**Steps**:
1. Tokenize each of the 5 fields (minute, hour, day, month, weekday)
2. Each field supports: `*`, `n`, `n-m` (range), `*/n` (step), `a,b,c` (list)
3. Implement `Next(t time.Time) time.Time` by incrementing fields from minute up
4. Write table-driven tests with at least 20 cases

---

### Phase 2 — Task Store with SQLite (Week 1)

```go
type Task struct {
    ID          string
    Type        string
    Payload     []byte         // JSON
    Status      TaskStatus     // pending / running / done / failed / dead
    Schedule    string         // cron expr or "" for one-shot
    NextRunAt   time.Time
    Attempts    int
    MaxAttempts int
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

Store interface:
```go
type Store interface {
    Enqueue(task *Task) error
    ClaimNext(workerID string) (*Task, error)    // SELECT ... FOR UPDATE equivalent
    MarkDone(taskID string) error
    MarkFailed(taskID string, err error) error
    Reschedule(taskID string, nextRun time.Time) error
    ListDead(limit int) ([]*Task, error)
    Retry(taskID string) error
}
```

**Key**: Use `database/sql` with a single file SQLite DB. Implement optimistic locking using
`status='pending' AND next_run_at <= now()` WHERE clause for `ClaimNext`.

---

### Phase 3 — Worker Pool (Week 2)

```go
pool := worker.NewPool(worker.Config{
    Concurrency: 10,
    PollInterval: 500ms,
    ShutdownTimeout: 30s,
})
pool.Register("send_email",    handlers.SendEmail)
pool.Register("gen_report",    handlers.GenerateReport)
pool.Register("charge_card",   handlers.ChargeCard)
pool.Start(ctx)
```

**Steps**:
1. N goroutines continuously call `store.ClaimNext()` in a polling loop
2. On claim, dispatch to the registered handler by `task.Type`
3. Track in-flight tasks with a `sync.WaitGroup`; on shutdown, drain gracefully
4. If handler returns error: increment attempts, if `>= maxAttempts` → `MarkDead`, else reschedule
5. Implement **jitter backoff**: `delay = base * 2^attempt + jitter(0..500ms)`

---

### Phase 4 — Scheduler Loop (Week 2)

A separate goroutine runs the scheduler that checks for recurring tasks:
1. Query store for tasks where `schedule != "" AND status = 'done' AND next_run_at <= now()`
2. For each, compute next fire time using `cron.Parse(task.Schedule).Next(now)`
3. Re-enqueue with updated `next_run_at`

**Distributed safety**: use a database-level `UPDATE ... WHERE status='done'` to
prevent double-scheduling when running multiple scheduler instances.

---

### Phase 5 — HTTP API + CLI (Week 3)

```bash
# Submit a one-shot task
scheduler submit --type send_email --payload '{"to":"user@example.com"}' --max-attempts 3

# Submit a recurring task
scheduler submit --type gen_report --cron "0 8 * * MON" --max-attempts 1

# View queue stats
scheduler status

# List dead-letter queue
scheduler dlq list

# Retry a dead task
scheduler dlq retry <task-id>
```

Implement a minimal HTTP server with these endpoints:
- `POST /tasks` — submit
- `GET /tasks/{id}` — status
- `POST /tasks/{id}/retry` — manual retry
- `GET /stats` — queue depth, worker utilization, DLQ size

---

## Acceptance Criteria

- [ ] 1,000 tasks/second throughput on a single node
- [ ] A crashed worker's claimed task is re-enqueued after a visibility timeout
- [ ] Recurring tasks fire within 2x `pollInterval` of their scheduled time
- [ ] No task is lost during graceful shutdown

---

## Stretch Goals

- Implement **at-most-once** semantics variant (idempotency key per task)
- Add **task chaining**: `OnSuccess` / `OnFailure` task IDs
- Build a TUI dashboard with Bubble Tea showing live queue stats
