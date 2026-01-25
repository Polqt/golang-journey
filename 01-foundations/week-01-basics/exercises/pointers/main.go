package main

import "fmt"

func increment(a *int) {
	*a++
}

func main() {
	var x int = 10
	var p *int = &x
	*p = 20

	fmt.Println("Value of x:", x)
	fmt.Println("Address of x:", p)
	fmt.Println("Value pointed to by p:", *p)

	var z *int = nil
	if z == nil {
		fmt.Println("Pointer is nill")
	}

	increment(&x)
	fmt.Println("x after increment:", x)
}