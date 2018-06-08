package main

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func main() {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "localhost"})
	if err != nil {
		panic(err)
	}

	// Delivery report handler for produced messages.
	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					fmt.Printf("Delivery failed: %v\n", ev.TopicPartition)
				} else {
					fmt.Printf("Delivered message to %v\n", ev.TopicPartition)
				}
			}
		}
	}()

	// Produce messages to topic (asynchronously)
	topic := "nstat.log"
	messages := []string{
		`{
    "mtype": "1",
    "oid": "100",
    "act": "add",
    "sid": "",
    "subkey": "",
    "value": "1",
    "created_at": "2018-05-25 08:02:58"
}`,
		`{
    "mtype": "11",
    "uid": "100",
    "nickname": "小新",
    "ip": "111.204.160.107",
    "created_at": "2018-05-25 08:02:58"
}`,
		`{
    "mtype": "41",
    "oid": "100",
    "act": "add",
    "filetype": "1",
    "filesize": "1024",
    "created_at": "2018-05-25 08:02:58"
}`,
	}
	for _, word := range messages {
		producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte(word),
		}, nil)
	}

	// Wait for message deliveries
	producer.Flush(15 * 1000)
}
