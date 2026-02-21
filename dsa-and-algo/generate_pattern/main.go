package main

import "fmt"

func generatePattern(n int) {
    if n >= 10 {
        fmt.Println("The number should be less than 10")
    } else if n <= 10 {
        for i := n; i >= 1; i-- {
            fmt.Println(i)
        }
    }

}

func main() {
	generatePattern(5)
}