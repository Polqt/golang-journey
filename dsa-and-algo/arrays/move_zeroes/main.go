package main

import "fmt"

func moveZeroes(nums []int) []int {
	if len(nums) == 0 {
		return nums
	}

	zero := 0

	for i := 0; i < len(nums); i++ {
		fmt.Printf("Checking number: %d at index %d\n", nums[i], i)
		if nums[i] != 0 {
			fmt.Printf("Moving %d to index %d\n", nums[i], zero)
			nums[zero] = nums[i]
			zero++
			fmt.Printf("Updated array: %v\n", nums)
		}
	}

	for zero < len(nums) {
		fmt.Printf("Filling zero at index %d\n", zero)
		nums[zero] = 0
		zero++
		fmt.Printf("Updated array: %v\n", nums)
	}

	return nums
}

func main() {
	fmt.Println(moveZeroes([]int{0, 1, 0, 3, 12}))
}
