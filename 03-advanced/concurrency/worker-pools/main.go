package main

/**
Let’s say you need to process a list of tasks, and each task
involves performing some I/O-bound or CPU-bound work. For simplicity,
we’ll simulate this with a function that takes time to complete each task.
Instead of spawning a new Go routine for each task (which can overwhelm the system),
we’ll use a worker pool to process these tasks efficiently.
*/

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Job represents the work that needs to be done
type Job struct {
	ID       int
	WorkTime time.Duration // How long the job takes to complete
}

// Worker represents a single worker in the pool
func worker(id int, jobs <-chan Job, wg *sync.WaitGroup) {
	defer wg.Done() // Signal the WaitGroup when done
	for job := range jobs {
		fmt.Printf("Worker %d started job %d\n", id, job.ID)
		time.Sleep(job.WorkTime) // Simulate doing the job
		fmt.Printf("Worker %d finished job %d\n", id, job.ID)
	}
}

func main() {
    const numWorkers = 4
    jobs := make(chan Job, 10)

    var wg sync.WaitGroup

    // Start workers
    for i := 1; i <= numWorkers; i++ {
        wg.Add(1)
        go worker(i, jobs, &wg)
    }

    // Create 10 jobs with random processing times
    for j := 1; j <= 10; j++ {
        workTime := time.Duration(rand.Intn(3)+1) * time.Second
        jobs <- Job{ID: j, WorkTime: workTime}
        fmt.Printf("Sent job %d to the job queue (work time: %v)\n", j, workTime)
    }

    close(jobs)

    wg.Wait()
    fmt.Println("All workers have finished")
}