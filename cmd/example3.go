package main

import (
	"fmt"
	"runtime"
	"sync"
)

var (
	counter int64
	wg      sync.WaitGroup
)

func addCount() {
	defer wg.Done()
	for count := 0; count < 2; count++ {
		value := counter
		// 当前goroutine从线程退出
		// 注意：当前goroutine从线程中退出后，下次执行的goroutine有可能是A也可能是B
		runtime.Gosched()
		value++
		counter = value
	}
}

func main() {
	wg.Add(2)
	go addCount() // G=A
	go addCount() // G=B
	wg.Wait()
	fmt.Printf("counter: %d\n", counter)
}

// Output:
// 输出2,3,4都有可能
