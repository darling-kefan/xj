package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/darling-kefan/xj/nstat/protocol"
)

var a protocol.Mtype = "hello"

var jsonStream string = `
{
    "mtype": "1",
    "oid": "100",
    "act": "add",
    "sid": "",
    "subkey": "",
    "value": "1",
    "created_at": "2018-05-25 08:02:58"
}
`

func DecodeLogMsg(r io.Reader) (x *protocol.LogMsg, err error) {
	x = new(protocol.LogMsg)
	if err = json.NewDecoder(r).Decode(x); err != nil {
		return
	}
	return
}

type CustomTime struct {
	time.Time
}

// type CustomTime time.Time
// 选择结构体方式还是类型别名方式,由于匿名结构体方式可以继承所有的time.Time方法,所以选择匿名结构体方式

const CustomTimeFormat = "2006-01-02 15:04:05"

func (ct *CustomTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		ct.Time = time.Time{}
		return
	}
	ct.Time, err = time.Parse(CustomTimeFormat, s)
	return
}

func (ct *CustomTime) MarshalJSON() ([]byte, error) {
	if ct.Time == (time.Time{}) {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", ct.Format(CustomTimeFormat))), nil
}

func main() {
	x, err := DecodeLogMsg(strings.NewReader(jsonStream))
	fmt.Printf("%#v %v\n", x, err)

	type T struct {
		CreatedAt *CustomTime `json:"created_at"`
	}
	t := new(T)
	var json2 string = `{"created_at": "2018-06-06 14:23:00"}`
	if err = json.Unmarshal([]byte(json2), t); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%#v\n", t.CreatedAt.Format(CustomTimeFormat))

	b, err := json.Marshal(t)
	fmt.Printf("%#v %v\n", string(b), err)
}
