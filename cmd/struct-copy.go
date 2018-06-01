package main

import (
	"log"
)

func main() {
	type T struct {
		Name string
	}
	t := T{Name: "tangshouqiang"}

	t1 := t
	t1.Name = "wanghao"

	log.Printf("%#v %#v\n", t, t1)
}

// Output:
// main.T{Name:"tangshouqiang"} main.T{Name:"wanghao"}
