package main

import (
	"fmt"
)

func main() {
	a := []int{1, 2, 3, 4}
	fmt.Println(a, a[:3])

	a = append(a, 11, 12, 13)
	fmt.Println(a)
}
