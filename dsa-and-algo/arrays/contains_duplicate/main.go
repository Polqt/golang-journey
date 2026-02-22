package main

import "fmt"

func containsDuplicate(nums []int) bool {
	if len(nums) == 0 {
		return false
	}

	m := make(map[int]bool)
	for _, num := range nums {
		fmt.Printf("Checking number: %d\n", num)
		if _, ok := m[num]; ok {
			fmt.Printf("Duplicate found: %d\n", num)
			return true
		}
		fmt.Printf("Adding %d to map\n", num)
		m[num] = true
	}
	fmt.Println("No duplicates found")
	return false
}

func main() {
	fmt.Println(containsDuplicate([]int{1, 2, 3, 1}))
}
