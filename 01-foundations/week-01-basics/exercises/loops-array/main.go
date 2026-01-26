package main

import (
	"fmt"
)

func main() {
	for i := 0; i <= 5; i++ {
		fmt.Println(i)
	}

	fmt.Println("")

	sum := 0
	for i := 1; i <= 10; i++ {
		sum += i
	}
	fmt.Println("Sum:", sum)

	fmt.Println("")

	for i := 1; i <= 10; i++ {
		if i%2 == 0 {
			fmt.Println(i, "is even")
		}
		continue
	}

	fmt.Println("")

	for i := 10; i >= 1; i-- {
		fmt.Println(i)
	}

	fmt.Println("")


	// Arrays and Slices
	cities := []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix"}
	for _, city := range cities {
		fmt.Printf("%s City\n", city)
	}

	favoriteCity := cities[2]
	fmt.Printf("My favorite city is: %s\n", favoriteCity)
	
	fmt.Println("")

	numbers := []int{10, 20, 30, 40, 50}
	numbers = append(numbers, 2, 20)
	fmt.Println("Numbers:", numbers)

	fmt.Println("")

	favoriteNumbers := []int{3, 7, 21, 42, 100}
	for index, number := range favoriteNumbers {
		fmt.Printf("Index: %d, Number: %d\n", index, number)
	}
	fmt.Printf("Length of array is: %d", len(favoriteNumbers))

	fmt.Println("")

	grid := [][]int{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}
	fmt.Println("")

	fmt.Printf("Grid has %d rows\n", len(grid))
	fmt.Printf("I will get 6: %d\n", grid[1][2])

	names := make([]string, 3 ,5)
	names[0] = "Alice"
	names[1] = "Bob"
	names[2] = "Charlie"


	for i, name := range names {
		fmt.Printf("Index: %d, Name: %s\n", i, name)
	}

	fmt.Println("")
	
	landscape := make([]int, 3, 5)
	fmt.Println("Length:", len(landscape))
	fmt.Println("Capacity:", cap(landscape))
}