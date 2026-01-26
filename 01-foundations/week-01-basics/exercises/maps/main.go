package main

import "fmt"

func main() {
	m := map[string]int{
        "Alice": 30,
        "Bob":   25,
    }

	m["Charlie"] = 35

	for name, age := range m {
		fmt.Printf("%s is %d years old\n", name, age)
	}

	fmt.Println("")

	delete(m, "Bob")

	if age, exists := m["Bob"]; exists {
		fmt.Printf("Bob is %d years old\n", age)
	} else {
		fmt.Println("Bob not found in the map")
	}

	fmt.Println()

	m["Diana"] = 28	
	fmt.Println("Updated map:", m)

	fmt.Println()

	for _, age := range m {
		fmt.Println("Age:", age)
	}
}