package nstat

// 日志分析器协程(Parser)：
//
// <- Parser.Inbound取出消息，解析成相应的消息类型；生成消息统计因子，并将其发往Parser.Outbound。

import (
	"log"
)

type Hub struct {
	Inbound  chan string
	Outbound chan *StatData
}

func (h *Hub) parse() {

}

func (h *Hub) run() {
	for {
		select {
		case <-h.Inbound:

		case <-stopCh:
			log.Println("Parser is existed.")
			return
		}
	}
}
