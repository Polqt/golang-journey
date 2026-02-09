package main

import (
	"fmt"
	"time"
)

func main() {
	ch := make(chan int)

	go sendData(ch)
	go receiveData(ch)

	time.Sleep(time.Second)


	// Channel Synchronization and Select
	/**
		The select statement in Go allows a goroutine to wait on 
		multiple communication operations simultaneously. 
		It adds sophistication to concurrent programming by enabling 
		goroutines to respond to multiple channels as they become ready.
	*/
	ch1 := make(chan string)
    ch2 := make(chan string)

    go func() {
        time.Sleep(time.Second)
        ch1 <- "Hello"
    }()

    go func() {
        time.Sleep(2 * time.Second)
        ch2 <- "World"
    }()

    select {
    case msg1 := <-ch1:
        fmt.Println(msg1)
    case msg2 := <-ch2:
        fmt.Println(msg2)
    }
}

func sendData(ch chan int) {
	ch <- 42
}

func receiveData(ch chan int) {
	data := <-ch
	fmt.Println("Received data:", data)
}
