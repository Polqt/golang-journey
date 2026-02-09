
# Go: Concurrency vs. Parallelism

Go is designed with concurrency as a core feature, making it easy to write programs that handle multiple tasks at once. It also supports parallelism on multi-core systems, thanks to its runtime scheduler.

---

## Concurrency

- **Concurrency** is about managing multiple tasks that may overlap in time, but don’t necessarily run at the exact same moment.
- Example: Downloading a file (I/O-bound) while letting the user interact with the UI (CPU-bound), switching between them as needed.
- In Go, concurrency is achieved with lightweight goroutines and channels, managed by the Go runtime.

## Parallelism

- **Parallelism** means actually running multiple tasks at the same time, usually on different CPU cores.
- Example: On a quad-core system, four goroutines can run truly simultaneously.
- Go uses as many CPU cores as are available by default. You can control this with the `GOMAXPROCS` setting (max OS threads for Go code).

---

## How Go Does It

- Goroutines: Lightweight, managed by Go’s runtime (not OS threads)
- Channels: Safe, easy communication between goroutines
- Scheduler: Automatically distributes goroutines across available CPU cores

> Concurrency is about structure; parallelism is about execution. Go gives you both, simply and powerfully.