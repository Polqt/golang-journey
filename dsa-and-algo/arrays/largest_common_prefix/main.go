package main

import "fmt"

func largestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}

	prefix := strs[0] // flower
	// Iterate through the remaining strings and compare with the current prefix
	for i := 1; i < len(strs) && prefix != ""; i++ { // flow
		j := 0
		for ; j < len(prefix) && j < len(strs[i]); j++ {
			if prefix[j] != strs[i][j] {
				break
			}
		}
		prefix = prefix[:j] 
	}

	return prefix
}

func main() {
	fmt.Println(largestCommonPrefix([]string{"flower", "flow"}))
}
