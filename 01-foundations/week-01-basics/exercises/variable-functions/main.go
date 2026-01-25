package main

import (
	"fmt"
	"strconv"
)

func main() {
	var result int
	var y, z int = 5, 10
	result = y + z

	grades := 85
	grade := ""

	const Pi = 3.14

	x := 42
	name := "Jepoy"

	fmt.Printf("%d + %d = %d\n", y, z, result)
	fmt.Println("Value of x is: " + strconv.Itoa(x))
	fmt.Printf("Hi there %s!\n", name)

	if x > 50 {
		fmt.Println("x is greater than 50")
	} else {
		fmt.Println("x is lesser than 50")
	}

	if grades >= 90 {
		grade = "A"
	} else if grades >= 80 {
		grade = "B"
	} else {
		grade = "C"
	}

	switch grade { 
	case "A":
		fmt.Println("You got an A!")
	case "B":
		fmt.Println("You got a B!")
	case "C":
		fmt.Println("You need to work harder!")
	default:
		fmt.Println("Invalid grade")
	}
}