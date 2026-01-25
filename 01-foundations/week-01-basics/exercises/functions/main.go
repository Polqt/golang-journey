package main

import (
	"fmt"
)

// Function to swap two strings
func swap(a, b string) (string, string) {
	return b, a
}

func greet(name string) {
	fmt.Printf("Hello, %s!\n", name)
}


// Variadic function to sum integers
func sum(nums ...int) int {
	total := 0
	for _, num := range nums {
		total += num
	}
	return total
}

func difference(x, y int) int {
	return x - y
}

// Higher-order function that takes a function as an argument
func applyOperation(x int, op func(int) int) int {
    return op(x)
}

func double(n int) int {
    return n * 2
}

// Defer example is like be the last to be executed
func example() {
    defer fmt.Println("This will be printed last")
    fmt.Println("This will be printed first")
}

func main() {
	num := 5

	a, b := swap("Hello", "World")
	fmt.Println(a, b)

	greet("Jepoy")

	total := sum(1, 2, 3, 4, 5)
	fmt.Println(total)

	result := difference(10, 5)	
	fmt.Println(result)

	doubled := applyOperation(num, double)
	fmt.Println(doubled)

	example()
}