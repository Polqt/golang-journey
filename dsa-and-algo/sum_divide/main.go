package main

import "fmt"

func sumAndDivide(num1, num2 int) float64 {
    sum := 0

    for i := num1; i <= num2; i++ {
        sum += i
    }
    result := float64(sum) / float64(num1 + num2)
    return result
}

func main() {
	fmt.Println(sumAndDivide(23, 123))
}