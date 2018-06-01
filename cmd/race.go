package main

import (
	"log"
)

func main() {
	//var m int
	//go func() {
	//	log.Printf("%#v\n", m)
	//}()
	//log.Printf("%#v\n", m)

	//go func() {
	//	m = m + 1
	//}()
	//log.Printf("%#v\n", m)

	type T struct {
		x int
	}
	t := T{x: 1}

	//go func() {
	//	log.Printf("%#v\n", t.x)
	//}()
	//log.Printf("%#v\n", t)

	//go func(t T) {
	//	t.x = t.x + 1
	//}(t)
	//log.Printf("%#v\n", t)

	//go func(t *T) {
	//	t.x = t.x + 1
	//}(&t)
	//log.Printf("%#v\n", t)

	go func() {
		t.x = t.x + 1
	}()
	log.Printf("%#v\n", t)
}
