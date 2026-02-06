package main

import (
	"fmt"
	"sync"
)

// Calculate the sum of numbers concurrently using goroutines and channels
func calculateSum(numbers []int, resultChan chan int, wg *sync.WaitGroup) {
	defer wg.Done()

	sum := 0
	for _, num := range numbers {
		sum += num
	}

	resultChan <- sum
}

func main() {
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	// Create a channel to receive results from goroutines
	resultChan := make(chan int)

	// WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Split the numbers into two halves
	mid := len(numbers) / 2

	// Launch two goroutines to calculate the sum of the first half and the second half of the numbers
	wg.Add(2)
	go calculateSum(numbers[:mid], resultChan, &wg)
	go calculateSum(numbers[mid:], resultChan, &wg)

	// Wait for both goroutines to finish
	wg.Wait()

	// CLose the result channel after all goroutines have finished sending results
	close(resultChan)

	// Collect results from the channel and calculate the final sum
	finalSum := 0
	for sum := range resultChan {
		finalSum += sum
	}

	fmt.Printf("The sum of numbers is: %d\n", finalSum)
}
