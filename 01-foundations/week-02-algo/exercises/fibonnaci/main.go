package main

import "fmt"

func main() {
	n := 10

	fib := make([]int, n)

	fib[0] = 0
	fib[1] = 1

	for i := 2; i < n; i++ {
		fib[i] = fib[i-1] + fib[i-2]
	}

	fmt.Println("Fibonacci sequence:")
	for _, num := range fib {
		fmt.Printf("%d ", num)
	}

	fmt.Println()
}