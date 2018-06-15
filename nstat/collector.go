package nstat

// 采集器协程(Collector)：
//
// 首先读取失败的消息，然后从上次结束进程记录的message_id处开始读取, 日志A -> processor.inbound。

// Golang Kafka Clients:
// https://cwiki.apache.org/confluence/display/KAFKA/Clients#Clients-Go(AKAgolang)
// Confluent-kafka-go VS Sarama-cluster:
// https://gist.github.com/savaki/a19dcc1e72cb5d621118fbee1db4e61f
// 选择使用Confluent-kafka-go, 完全支持Kafka0.9及以上.

import (
	"bytes"
	"log"
	"strconv"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/darling-kefan/xj/config"
	"github.com/darling-kefan/xj/nstat/protocol"
	"github.com/segmentio/ksuid"
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
	log.Println("Collector start...")
	defer wg.Done()

	groupID := ksuid.New().String()
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"client.id":                       "nstat",
		"bootstrap.servers":               config.Config.Kafka.Servers,
		"group.id":                        groupID,
		"session.timeout.ms":              6000, // TODO 6s还是6000s???
		"enable.auto.commit":              false,
		"go.events.channel.enable":        true,
		"go.application.rebalance.enable": true,
		"default.topic.config":            kafka.ConfigMap{"auto.offset.reset": "earliest"},
	})
	if err != nil {
		log.Println("debug............", err)
		// 通知其它goroutine退出
		toStop <- "stop"
		return
	}
	defer consumer.Close()

	consumer.SubscribeTopics([]string{"nstat.login.log", "nstat.courseware.log", "nstat.log"}, nil)

loop:
	for {
		select {
		case m, ok := <-consumer.Events():
			if !ok {
				// TODO 如何处理
				log.Println("unexpected eof")
				break loop
			}

			switch event := m.(type) {
			case kafka.AssignedPartitions:
				consumer.Assign(event.Partitions)
			case kafka.PartitionEOF:
				// nop
			case kafka.RevokedPartitions:
				consumer.Unassign()
			case *kafka.Message:
				// 解析message
				log.Printf("%#v, %s\n", string(event.Value), event.TopicPartition)

				logMsg, err := protocol.DecodeLogMsg(bytes.NewReader(event.Value))
				if err != nil {
					log.Println("Failed to decode json: ", err)
					break
				}

				logMsg.Topic = *event.TopicPartition.Topic
				logMsg.Partition = strconv.Itoa(int(event.TopicPartition.Partition))
				logMsg.Offset = event.TopicPartition.Offset.String()
				c.p.inbound <- logMsg
			default:
				log.Println(event)
			}
		case <-stopCh:
			log.Println("Collector quit...")
			return
		}
	}

	// 通知其它goroutine退出
	toStop <- "stop"
}
