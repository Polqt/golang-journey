package main

import "fmt"

func plusOne(digits []int) []int {
	if len(digits) == 0 {
		return []int{1}
	}

	for i := len(digits) - 1; i >= 0; i-- { // if digits = [1, 2, 3]  ti kay 3 ang value ni i, 3 then - 1 which is 2. so ang i is 2.
		fmt.Println("Processing digit at index", i, ":", digits[i]) 
		if digits[i] < 9 {
			fmt.Printf("Digit at index %d is less than 9, incrementing it\n", i)
			digits[i]++
			return digits
		} else {
			fmt.Printf("Digit at index %d is 9, setting to 0\n", i)
			digits[i] = 0
		}
	}
	digits = append([]int{1}, digits...)
	return digits
}

func main() {
	fmt.Println(plusOne([]int{1, 2, 3}))
}
