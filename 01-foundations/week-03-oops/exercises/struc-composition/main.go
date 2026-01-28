package main

import "fmt"

type Person struct {
	name string
	age  int
}

func main() {
	p := Person{
		name: "John",
		age:  25,
	}

	p1 := Person{
		name: "Alice",
		age:  30,
	}
	fmt.Printf("%s: %d", p1.name, p1.age)

	fmt.Println()

	modifyPerson(&p)
	println("Modified name:", p.name)

	
}

func modifyPerson(p *Person) {
	p.name = "Bob"
}