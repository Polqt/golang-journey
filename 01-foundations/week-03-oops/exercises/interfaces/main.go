package main

import "fmt"

// Define Interface
type Stringer interface {
	String() string
}

// Implement interface
type Person struct {
	Name string
}

func (p Person) String() string {
	return p.Name
}

// Using Interface
func PrintString(s Stringer) {
	fmt.Println(s.String())
}

// Empty Interface
func PrintAnything(v interface{}) {
	fmt.Println(v)
}

// Interface Compliance
type Shape interface {
	Area() float64
}

type Circle struct {
	Radius float64
}

func (c Circle) Area() float64 {
	return 3.14 * c.Radius * c.Radius
}

func main() {
	// Type Assertions
	// Type assertions are used to retrieve the dynamic type of an interface. 
	// They are useful when you need to work with the underlying type of an interface value.
	var i interface{} = "hello"

	p := Person{Name: "Ben"}
	PrintString(p)
	fmt.Println("")

	PrintAnything("Hi")
	PrintAnything(12345)
	PrintAnything([]int{1, 2, 3})

	fmt.Println("")

	s, ok := i.(string) 
	if ok {
		fmt.Println("String value:",s)
	} else {
		fmt.Println("Not a string")
	}

	fmt.Println("")
	c := Circle{Radius: 5}
	fmt.Printf("Circle area: %.2f\n", c.Area())
}