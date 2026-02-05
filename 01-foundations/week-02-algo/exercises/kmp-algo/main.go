package main

import "fmt"

// Compute the longest prefix suffix (lps) array for the pattern
func computeLPS(pattern string) []int {
    length := len(pattern)
    lps := make([]int, length)
    
    j := 0 // Length of the previous longest prefix suffix
    
    for i := 1; i < length; {
        if pattern[i] == pattern[j] {
            j++
            lps[i] = j
            i++
        } else {
            if j != 0 {
                j = lps[j-1]
            } else {
                lps[i] = 0
                i++
            }
        }
    }
    
    return lps
}

// Search for the pattern in the text using the KMP algorithm
func searchKMP(text, pattern string) []int {
    result := make([]int, 0)
    textLength := len(text)
    patternLength := len(pattern)
    lps := computeLPS(pattern)
    
    i, j := 0, 0 // i for text, j for pattern
    
    for i < textLength {
        if pattern[j] == text[i] {
            i++
            j++
        }
        if j == patternLength {
            // Pattern found, add the starting index to the result
            result = append(result, i-j)
            j = lps[j-1]
        } else if i < textLength && pattern[j] != text[i] {
            if j != 0 {
                j = lps[j-1]
            } else {
                i++
            }
        }
    }
    
    return result
}

func main() {
    // Example: Search for the pattern "abc" in the text "ababcabcabcabc"
    text := "ababcabcabcabc"
    pattern := "abc"
    
    indices := searchKMP(text, pattern)
    
    if len(indices) > 0 {
        fmt.Printf("Pattern found at positions: %v\n", indices)
    } else {
        fmt.Println("Pattern not found in the text.")
    }
}