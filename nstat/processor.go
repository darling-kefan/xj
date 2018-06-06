package nstat

// 日志处理器协程(Processor)：
//
// <- processor.inbound取出消息，解析成相应的消息类型；生成消息统计因子，并将其发往processor.outbound。

import (
	"encoding/json"
	"log"
	"sync"
)

type processor struct {
	inbound  chan string
	outbound chan *StatData
}

func newProcessor() *processor {
	return processor{
		inbound:  make(chan string),
		outbound: make(chan *StatData),
	}
}

func (p *processor) handle(message string) *StatData {

}

func (p *processor) run(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case message <- h.Inbound:

		case <-stopCh:
			log.Println("Processor quit.")
			return
		}
	}
}
