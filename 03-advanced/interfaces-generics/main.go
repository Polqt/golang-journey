package main


func AddInts(a, b int) int {
	return a + b
}

func AddFloats(a, b float64) float64 {
	return a + b
}

type Number interface {
	int | float64
}

type Pair[T any] struct {
	First, Second T
}

func Add[T Number](a, b T) T {
	return a + b
}

func main() {
	

}