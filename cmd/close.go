package main

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"
)

func main() {
	ch := make(chan int, 2000)

	var wg sync.WaitGroup

	go func() {
		for i := 0; i < 100; i++ {
			ch <- i
		}
		close(ch)
		fmt.Printf("[main] line:22 producer end\n")
	}()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go work(&wg, ch)
	}
	wg.Wait()
	fmt.Println("exit")
}

func work(wg *sync.WaitGroup, ch chan int) {
	gid := getGID()
	defer wg.Done()
	for {
		time.Sleep(100 * time.Millisecond)
		select {
		case data, ok := <-ch:
			fmt.Printf("[%d] line:39 %#v %#v\n", gid, data, ok)
		default:
			fmt.Printf("[%d] line:41 not data\n", gid)
		}

		item, ok := <-ch
		if !ok {
			fmt.Printf("[%d] line:46 %#v %#v\n", gid, item, ok)
			break
		} else {
			fmt.Printf("[%d] line:49 %#v %#v\n", gid, item, ok)
		}
	}
	fmt.Printf("[%d] line:52 worker exit\n", gid)
}

// Get Goroutine ID
func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
