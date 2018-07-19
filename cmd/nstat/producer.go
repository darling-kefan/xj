package main

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func main() {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "127.0.0.1:22190"})
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
	// 用户机构消息
	msg_mtype1 := []string{`{
    "mtype": "1",
    "oid": "1",
    "act": "add",
    "sid": "1000000",
    "subkey": "",
    "value": "1",
    "created_at": "2018-07-19 15:53:58"
}`,
		`{
    "mtype": "1",
    "oid": "1",
    "act": "add",
    "sid": "1000001",
    "subkey": "",
    "value": "1",
    "created_at": "2018-07-19 15:53:58"
}`,
		`{
    "mtype": "1",
    "oid": "1",
    "act": "add",
    "sid": "1000002",
    "subkey": "",
    "value": "1",
    "created_at": "2018-07-19 15:53:58"
}`,
		`{
    "mtype": "1",
    "oid": "1",
    "act": "add",
    "sid": "1000003",
    "subkey": "",
    "value": "1",
    "created_at": "2018-07-19 15:53:58"
}`}
	// 用户登录消息
	msg_mtype11 := []string{`{
    "mtype": "11",
    "oid": "1",
    "uid": "8",
    "nickname": "老师3",
    "ip": "111.204.160.107",
    "created_at": "2018-07-19 18:02:58"
}`, `{
    "mtype": "11",
    "oid": "1",
    "uid": "7",
    "nickname": "laoshi2",
    "ip": "111.204.160.107",
    "created_at": "2018-07-19 18:02:58"
}`, `{
    "mtype": "11",
    "oid": "1",
    "uid": "1",
    "nickname": "laoshi1",
    "ip": "111.204.160.107",
    "created_at": "2018-07-19 18:02:58"
}`, `{
    "mtype": "11",
    "oid": "1",
    "uid": "100",
    "nickname": "小新",
    "ip": "111.204.160.107",
    "created_at": "2018-07-19 18:02:58"
}`,
		`{
    "mtype": "11",
    "oid": "1",
    "uid": "102",
    "nickname": "小白",
    "ip": "180.201.0.1",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "11",
    "oid": "1",
    "uid": "103",
    "nickname": "小黑",
    "ip": "222.173.0.1",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "11",
    "oid": "1",
    "uid": "104",
    "nickname": "小红",
    "ip": "27.192.0.1",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "11",
    "oid": "1",
    "uid": "105",
    "nickname": "小紫",
    "ip": "123.128.0.1",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "11",
    "oid": "1",
    "uid": "106",
    "nickname": "小蓝",
    "ip": "202.102.128.1",
    "created_at": "2018-07-19 08:02:58"
}`,
	}
	// 课程消息
	msg_mtype21 := []string{`{
    "mtype": "21",
    "oid": "1",
    "act": "add",
    "sid": "100",
    "subkey": "",
    "value": "1",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "21",
    "oid": "1",
    "act": "add",
    "sid": "101",
    "subkey": "",
    "value": "1",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "21",
    "oid": "1",
    "act": "add",
    "sid": "102",
    "subkey": "",
    "value": "1",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "21",
    "oid": "1",
    "act": "add",
    "sid": "103",
    "subkey": "",
    "value": "1",
    "created_at": "2018-07-19 08:02:58"
}`}
	// 课程用户消息
	msg_mtype22 := []string{`{
    "mtype": "22",
    "oid": "1",
    "act": "add",
    "sid": "100",
    "subkey": "1000000",
    "value": "1",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "22",
    "oid": "1",
    "act": "add",
    "sid": "100",
    "subkey": "1000000",
    "value": "1",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "22",
    "oid": "1",
    "act": "add",
    "sid": "100",
    "subkey": "1000000",
    "value": "1",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "22",
    "oid": "1",
    "act": "add",
    "sid": "100",
    "subkey": "1000000",
    "value": "1",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "22",
    "oid": "1",
    "act": "add",
    "sid": "100",
    "subkey": "1000000",
    "value": "1",
    "created_at": "2018-07-19 08:02:58"
}`}
	// 课件消息
	msg_mtype41 := []string{`{
    "mtype": "41",
    "oid": "1",
    "act": "add",
    "filetype": "1",
    "filesize": "131024",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "41",
    "oid": "1",
    "act": "add",
    "filetype": "2",
    "filesize": "12024",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "41",
    "oid": "1",
    "act": "add",
    "filetype": "3",
    "filesize": "144024",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "41",
    "oid": "1",
    "act": "add",
    "filetype": "4",
    "filesize": "111024",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "41",
    "oid": "1",
    "act": "add",
    "filetype": "5",
    "filesize": "111024",
    "created_at": "2018-07-19 08:02:58"
}`,
	}
	// 订单用户消息
	msg_mtype51 := []string{`{
    "mtype": "51",
    "oid": "1",
    "act": "add",
    "sid": "10000111",
    "subkey": "5",
    "value": "120",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "51",
    "oid": "1",
    "act": "add",
    "sid": "10000111",
    "subkey": "5",
    "value": "100",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "51",
    "oid": "1",
    "act": "done",
    "sid": "10000111",
    "subkey": "5",
    "value": "215",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "51",
    "oid": "1",
    "act": "done",
    "sid": "10000111",
    "subkey": "5",
    "value": "512",
    "created_at": "2018-07-19 08:02:58"
}`,
		`{
    "mtype": "51",
    "oid": "1",
    "act": "done",
    "sid": "10000111",
    "subkey": "5",
    "value": "111",
    "created_at": "2018-07-19 08:02:58"
}`}

	messages := make([]string, 0)
	messages = append(messages, msg_mtype1...)
	messages = append(messages, msg_mtype11...)
	messages = append(messages, msg_mtype21...)
	messages = append(messages, msg_mtype22...)
	messages = append(messages, msg_mtype41...)
	messages = append(messages, msg_mtype51...)
	fmt.Println(messages)
	for _, word := range messages {
		producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte(word),
		}, nil)
	}

	// Wait for message deliveries
	producer.Flush(15 * 1000)
}
