package main

import "fmt"

func main() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic:", r)
			}
		}()
		panic("something went wrong")
	}()
}
