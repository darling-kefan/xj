package main

import (
	"fmt"

	"github.com/ipipdotnet/datx-go"
)

func main() {
	city, err := datx.NewCity("/home/shouqiang/goyards/src/github.com/darling-kefan/xj/17monipdb.datx")
	if err == nil {
		fmt.Println(city.Find("8.8.8.8"))
		fmt.Println(city.Find("128.8.8.8"))
		fmt.Println(city.Find("255.255.255.255"))
		districts, err := city.Find("202.168.153.237")
		fmt.Printf("%#v %#v\n", districts, err)
		districts, err = city.Find("111.204.160.107")
		fmt.Printf("%#v %#v\n", districts, err)
	}
}
