package main

import (
    "fmt"
    "sync"
)

var (
    counter int
    mutex   sync.Mutex
)

func increment() {
    for i := 0; i < 1000; i++ {
        mutex.Lock()   // Acquire the lock before modifying the counter
        counter++
        mutex.Unlock() // Release the lock after modification
    }
}

func main() {
    var wg sync.WaitGroup
    wg.Add(2)

    go func() {
        defer wg.Done()
        increment()
    }()

    go func() {
        defer wg.Done()
        increment()
    }()

    wg.Wait()
    fmt.Println("Final Counter:", counter)
}