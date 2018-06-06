package nstat

// 采集器协程(Collector)：
//
// 首先读取失败的消息，然后从上次结束进程记录的message_id处开始读取, 日志A -> processor.inbound。

import (
	"log"
	"sync"
)

type collector struct {
	p *processor
}

func newCollector(p *processor) *collector {
	return &collector{
		p: p,
	}
}

func (c *collector) run(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		msg := "hello world"
		select {
		case c.p.inbound <- msg:

		case <-stopCh:
			log.Println("Consumer quit.")
			return
		}
	}
}
