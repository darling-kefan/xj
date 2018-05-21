package main

import (
	"encoding/json"
	"log"
	"time"
)

func main() {
	var jsonRaw = []byte(`{"created_at": "2018-03-05T12:30:00Z"}`)

	type Response struct {
		CreatedAt time.Time `json:"created_at"`
	}
	var resp Response

	err := json.Unmarshal(jsonRaw, &resp)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(resp)
}
