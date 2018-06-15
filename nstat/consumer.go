package nstat

// 消费者协程(Consumer)： 该协程可根据具体情况开启多个。
//
// <- processor.outbound取出消息统计因子，每10个消息统计因子作为参数请求一次api。
// 重拾3次，若3次均失败则将message_id持久化并发通知结束进程。退出协程时，
// 将消费的最新(大)message_id记录到main goroutine的最新message_id变量存储。

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/darling-kefan/xj/config"
	"github.com/darling-kefan/xj/nstat/protocol"
)

// Get Goroutine ID
func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

type consumer struct {
	p *processor
}

func newConsumer(p *processor) *consumer {
	return &consumer{
		p: p,
	}
}

func (c *consumer) run(wg *sync.WaitGroup) {
	// Goroutine ID
	goroutineId := getGID()

	log.Printf("Consumer[%v] start...\n", goroutineId)
	defer wg.Done()

	// 1毫秒读取一次通道信息, 以同时处理多个消息(传递给接口), 从而提高效率.
	ticker := time.NewTicker(1 * time.Millisecond)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-ticker.C:
			// 一次性最多处理10条消息
			n := 10
			factors := make([]*protocol.StatFactor, 0)
		inner:
			for i := 0; i < n; i++ {
				select {
				case statData := <-c.p.outbound:
					factors = append(factors, statData.Factors...)
				default:
					break inner
					//如果c.p.outbound中无数据, 则执行default.
					//log.Printf("Consumer[%v] no data in outbound.\n", goroutineId)
				}
			}

			// 该种写法不妥当, 因为len反映此时此刻通道里的数据,
			// 如果是多个goroutine的话, 会堵塞在case语句里面导致进程退出不了.
			//n, length := 10, len(c.p.outbound)
			//if length < 10 && length > 0 {
			//	n = length
			//}
			//for i := 0; i < n; i++ {
			//	statData = <-c.p.outbound
			//	factors = append(factors, statData.Factors...)
			//}

			// 请求接口,重试3次，同步到ssdb
			attempt := 1
			for {
				if err := CommitFactors(factors); err != nil {
					log.Printf("Consumer[%v] failed to commit factors, error: %s\n", goroutineId, err)
					attempt = attempt + 1
				} else {
					break
				}
				if attempt > 3 {
					// 通知关闭其它协程
					toStop <- "stop"
					goto end
				}
			}
		case <-stopCh:
			break loop
		}
	}

	// 执行如下代码，用于防止c.p.outbound通道里仍有未处理数据
loop2:
	for statData := range c.p.outbound {
		factors := make([]*protocol.StatFactor, 0)
		factors = append(factors, statData.Factors...)

		// 请求接口,重试3次，同步到ssdb
		attempt := 1
		for {
			if err := CommitFactors(factors); err != nil {
				log.Printf("Consumer[%v] failed to commit factors, error: %s\n", goroutineId, err)
				attempt = attempt + 1
			} else {
				break
			}
			if attempt > 3 {
				// 通知关闭其它协程
				toStop <- "stop"
				break loop2
			}
		}
	}

end:
	log.Printf("Consumer[%v] quit...\n", goroutineId)
}

// 将统计因子持久化到ssdb
func CommitFactors(factors []*protocol.StatFactor) error {
	if len(factors) == 0 {
		return nil
	}

	jsonStream, err := json.Marshal(factors)
	if err != nil {
		return err
	}
	log.Printf("%s\n", string(jsonStream))

	// 请求接口,同步到ssdb
	url := fmt.Sprintf("%s/v1/stat/add", config.Config.Common.StatHost)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStream))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("debug.................", err)
		return err
	}

	log.Println(string(body))
	type Data struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	var data Data
	if err = json.Unmarshal(body, &data); err != nil {
		log.Println("debug...............", err)
		return err
	}
	if data.Errcode != 0 {
		return errors.New(data.Errmsg)
	}

	return nil
	// TODO 记录未被处理的消息
	// --------------------------------
}
