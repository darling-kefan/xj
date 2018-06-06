package nstat

// 消费者协程(Consumer)： 该协程可根据具体情况开启多个。
//
// <- processor.outbound取出消息统计因子，每10个消息统计因子作为参数请求一次api。
// 重拾3次，若3次均失败则将message_id持久化并发通知结束进程。退出协程时，
// 将消费的最新(大)message_id记录到main goroutine的最新message_id变量存储。

import (
	"log"
	"sync"
)

type consumer struct {
	p *processor
}

func newConsumer(p *processor) *consumer {
	return &consumer{
		p: p,
	}
}

func (c *consumer) run(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-c.p.inbound:

		case <-stopCh:
			log.Println("Consumer quit.")
			return
		}
	}
}
