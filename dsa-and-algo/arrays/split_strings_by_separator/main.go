package main

import "fmt"

func splitWordsBySeparator(words []string, separator byte) []string {
	if len(words) == 0 {
		return []string{}
	}

	var result []string
	for _, word := range words {
		start := 0
		for i := 0; i < len(word); i++ {
			if word[i] == separator {
				if start < i {
					result = append(result, word[start:i])
				}
				start = i + 1
				fmt.Printf("Found separator at index %d, current result: %v\n", i, result)
			}
		}
		if start < len(word) {
			result = append(result, word[start:])
			fmt.Printf("Adding last segment: %s, current result: %v\n", word[start:], result)
		}

	}

	return result
}

func main() {
	fmt.Println(splitWordsBySeparator([]string{"one.two.three", "four.five", "six"}, '.'))
}
