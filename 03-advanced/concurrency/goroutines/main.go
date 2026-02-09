package main

import (
	"fmt"
	"time"
)

func sayHello() {
	fmt.Println("Hello, World!")
}

func main() {
	go sayHello() // Run sayHello concurrently
	time.Sleep(1 * time.Second)

}
