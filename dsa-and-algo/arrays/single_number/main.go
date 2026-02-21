package main

import "fmt"

func singleNumber(nums []int) int {
	if len(nums) == 0 {
		return 0
	}

	result := 0
	for _, num := range nums { 
		result ^= num

	}
	return result

}

func main() {
	fmt.Println(singleNumber([]int{4, 1, 2, 1, 2}))
}
