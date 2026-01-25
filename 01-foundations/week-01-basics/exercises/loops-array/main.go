package main

import "fmt"

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

	cities := []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix"}
	for _, city := range cities {
		fmt.Printf("%s City\n", city)
	}

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
	fmt.Printf("I will get 6: %d", grid[1][2])
}