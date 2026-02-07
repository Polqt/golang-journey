package main

import (
	"fmt"
	"sort"
)

func greedyCoinExchange(coins []int, amount int) []int {
	sort.Sort(sort.Reverse(sort.IntSlice(coins)))

	change := make([]int, 0)

	for _, coin := range coins {
		for amount >= coin {
			amount -= coin
			change = append(change, coin)
		}
	}

	if amount == 0 {
		return change
	}

	return []int{}
}

func main() {
	coins := []int{1, 5, 10, 25}
	amount := 63
	change := greedyCoinExchange(coins, amount)

	if len(change) > 0 {
		fmt.Printf("Change for %d cents: %v\n", amount, change)
	} else {
		fmt.Println("Change cannot be made with the given coins.")
	}
}
