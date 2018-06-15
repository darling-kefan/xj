package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Record struct {
	AuthorRaw json.RawMessage `json:"author"`
	Title     string          `json:"title"`
	URL       string          `json:"url"`

	AuthorEmail string
	AuthorID    uint64
}

func Decode(r io.Reader) (x *Record, err error) {
	x = new(Record)
	if err = json.NewDecoder(r).Decode(x); err != nil {
		return
	}
	var s string
	if err = json.Unmarshal(x.AuthorRaw, &s); err == nil {
		x.AuthorEmail = s
		return
	}
	var n json.Number
	if err = json.Unmarshal(x.AuthorRaw, &n); err == nil {
		nn, _ := n.Int64()
		x.AuthorID = uint64(nn)
	}
	return
}

func main() {
	jsonStream := `{"author": 99999999999999999999999999999999, "title": "lession 1", "url": "https://www.sina.com.cn"}`
	x, err := Decode(strings.NewReader(jsonStream))
	fmt.Printf("%v: %s : %s: %s: %v\n", x.AuthorID, x.AuthorEmail, x.Title, x.URL, err)

	type Test struct {
		A     json.RawMessage `json:"a"`
		vid   uint64
		vname string
	}
	var a Test
	err = json.Unmarshal([]byte(`{"a": "123"}`), &a)
	fmt.Printf("%#v %#v %#v\n", a, err, err == nil)

	var m string
	if err = json.Unmarshal(a.A, &m); err == nil {
		a.vname = m
	}
	var n float64
	if err = json.Unmarshal(a.A, &n); err == nil {
		a.vid = uint64(n)
	}
	fmt.Printf("%#v\n", a)
}
