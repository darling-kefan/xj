package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Record struct {
	Author Author `json:"author"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

type Author struct {
	ID    uint64 `json:"id"`
	Email string `json:"email"`
}

// Used to avoid recursion in UnmarshalJSON beblow.
type author Author

func (a *Author) UnmarshalJSON(b []byte) (err error) {
	j, s, n := author{}, "", uint64(0)
	if err = json.Unmarshal(b, &j); err == nil {
		*a = Author(j)
		return
	}
	if err = json.Unmarshal(b, &s); err == nil {
		a.Email = s
		return
	}
	if err = json.Unmarshal(b, &n); err == nil {
		a.ID = n
	}
	return
}

//func Decode(r io.Reader) (x *Record, err error) {
//	x = new(Record)
//	err = json.NewDecoder(r).Decode(x)
//	return
//}

type Records []Record

func Decode(r io.Reader) (x Records, err error) {
	err = json.NewDecoder(r).Decode(&x)
	return
}

func main() {
	jsonStream := `
[{
  "author": "attila@attilaolah.eu",
  "title":  "My Blog",
  "url":    "http://attilaolah.eu"
}, {
  "author": 1234567890,
  "title":  "Westartup",
  "url":    "http://www.westartup.eu"
}, {
  "author": {
    "id":    1234567890,
    "email": "nospam@westartup.eu"
  },
  "title":  "Westartup",
  "url":    "http://www.westartup.eu"
}]`

	records, err := Decode(strings.NewReader(jsonStream))
	fmt.Printf("%#v %v\n", records, err)
}
