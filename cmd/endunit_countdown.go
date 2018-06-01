package main

// 1. Scan Redis KEY="nc:chan:unit:run:{unitid}", 定时执行Scan, 并维护一个监控表
//
// 2. Start a goroutine to monitor unit
// > 监测KEY是否存在，如果不存在则表示单元结束，通知云中控结束单元，通知Canvas结束单元；
// > 将该单元加入到监控表, 单元结束后从监控表移除；
//
// 注意点：
// > C-c结束进程，需终止所有goroutine
//
// go run endunit_countdown.go --config_file_path="/home/shouqiang/go/src/github.com/darling-kefan/xj"

import (
	//"bytes"
	"context"
	"encoding/json"
	//"errors"
	"flag"
	"fmt"
	//"io/ioutil"
	"log"
	//"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/websocket"

	"github.com/darling-kefan/xj/config"
	"github.com/darling-kefan/xj/helper"
)

const (
	scanPrefix    string        = "nc:chan:unit:run:"
	scanPeriod    time.Duration = 5 * time.Second
	monitorPeriod time.Duration = 1 * time.Second
)

// 管理所有监测中的单元
type Bus struct {
	unitscenes map[string]bool
	// 读写锁
	mutex *sync.RWMutex
	// Worker goroutine tells the main goroutine quited
	done chan struct{}
	// Main goroutine informs worker goroutine to end work
	kill chan struct{}
}

// 实例化Bus
func NewBus() *Bus {
	return &Bus{
		unitscenes: make(map[string]bool),
		// 注意：此处一定要初始化
		mutex: new(sync.RWMutex),
		done:  make(chan struct{}),
		kill:  make(chan struct{}),
	}
}

// 判断该unitscene是否在监测中
func (b *Bus) In(unitscene string) bool {
	b.mutex.RLock()
	_, isIn := b.unitscenes[unitscene]
	b.mutex.RUnlock()
	return isIn
}

// 将单元添加到监控中
func (b *Bus) Add(unitscene string) {
	b.mutex.Lock()
	b.unitscenes[unitscene] = true
	b.mutex.Unlock()
}

// 将单元从监控中移除
func (b *Bus) Remove(unitscene string) {
	b.mutex.Lock()
	delete(b.unitscenes, unitscene)
	b.mutex.Unlock()
}

// 扫描redis keys获得所有待监测key
func scan(ctx context.Context) ([]string, error) {
	redconn := ctx.Value("redisPool").(*redis.Pool).Get()
	defer redconn.Close()

	iter := 0
	match := scanPrefix + "*"
	var keys []string
	for {
		if bulks, err := redis.MultiBulk(redconn.Do("SCAN", iter, "MATCH", match, "COUNT", 1000)); err != nil {
			return nil, err
		} else {
			iter, _ = redis.Int(bulks[0], nil)
			partkeys, _ := redis.Strings(bulks[1], nil)
			keys = append(keys, partkeys...)
		}
		// iter == 0标识迭代结束
		if iter == 0 {
			break
		}
	}
	return keys, nil
}

type EndUnitPacket struct {
	Act  string `json:"act"`
	From string `json:"from"`
	Msg  struct {
		Stat string `json:"stat"`
	} `json:"msg"`
}

// 结束单元
func endUnit(ctx context.Context, bus *Bus, unitid string, isEndCloud bool) error {
	// redis实例
	redconn := ctx.Value("redisPool").(*redis.Pool).Get()

	// 获取token
	token, err := helper.AccessToken(redconn, "client_credentials", nil)
	if err != nil {
		return err
	}

	if !isEndCloud {
		// 通知云中控单元结束
		wsapi := strings.Replace(strings.Replace(config.Config.Cc.Wsapi, ":unit_id", unitid, -1), ":token", token, -1)
		c, _, err := websocket.DefaultDialer.Dial(wsapi, nil)
		if err != nil {
			log.Fatal("dial: ", err)
		}
		defer c.Close()

		endunitPacket := EndUnitPacket{
			Act:  "12",
			From: "0:endunit_countdown.go",
			Msg: struct {
				Stat string `json:"stat"`
			}{
				Stat: "2",
			},
		}
		b, err := json.Marshal(endunitPacket)
		if err != nil {
			return err
		}
		err = c.WriteMessage(websocket.TextMessage, []byte(b))
		if err != nil {
			log.Fatal(err)
		}
	}

	// 通知Canvas单元结束
	//api := config.Config.Api.Domain + "/v1/units/:unit_id/status?token=:token"
	//api = strings.Replace(strings.Replace(api, ":unit_id", unitid, -1), ":token", token, -1)
	//payload := []byte(`{"status": 2}`)
	//req, err := http.NewRequest("POST", api, bytes.NewBuffer(payload))
	//if err != nil {
	//	return err
	//}
	//req.Header.Set("Content-Type", "application/json")
	//client := &http.Client{}
	//resp, err := client.Do(req)
	//if err != nil {
	//	return err
	//}
	//body, err := ioutil.ReadAll(resp.Body)
	//log.Println(api, string(body))
	//if err != nil {
	//	return err
	//}
	//defer resp.Body.Close()
	//type Response struct {
	//	Errcode int    `json:"errcode"`
	//	Errmsg  string `json:"errmsg"`
	//}
	//var response Response
	//err = json.Unmarshal(body, &response)
	//if err != nil {
	//	return err
	//}
	//if response.Errcode != 0 {
	//	return errors.New(response.Errmsg)
	//}
	return nil
}

// 监控单元是否结束
func monitor(ctx context.Context, bus *Bus) {
	// redis实例
	redconn := ctx.Value("redisPool").(*redis.Pool).Get()

	ticker := time.NewTicker(monitorPeriod)
	defer func() {
		log.Println("monitor goroutine is exit!")
		ticker.Stop()
		close(bus.done)
	}()

	for {
		select {
		case <-bus.kill:
			return
		case <-ticker.C:
			log.Println("monitor goroutine, debug..........", bus.unitscenes)
			for unitscene, _ := range bus.unitscenes {
				usparts := strings.Split(unitscene, ":")
				unitid, sceneid := usparts[0], usparts[1]

				// 判断云中控是否已经结束课程
				isEndCloud := false
				scenekey := fmt.Sprintf("nc:unit:scene:%s:%s", unitid, sceneid)
				scenebytes, _ := redis.Bytes(redconn.Do("GET", scenekey))
				if len(scenebytes) > 0 {
					var sceneinfo struct {
						Endtime float64 `json:"end_time"`
					} = struct {
						Endtime float64 `json:"end_time"`
					}{}
					err := json.Unmarshal(scenebytes, &sceneinfo)
					if err != nil {
						log.Println(err)
					}
					if sceneinfo.Endtime > 0 {
						isEndCloud = true
					}
				}

				// 此处是根据时间判断，改成ttl方式根据已过时间来判断
				// 3600-ttl >= 3600，第一个3600在lua中确定的，第二个3600由业务确定
				runkey := scanPrefix + unitscene
				ttl, _ := redis.Int(redconn.Do("TTL", runkey))
				if ttl == -2 || (ttl >= 0 && 3600-ttl >= 3600) {
					// 在监控中心移除该单元
					bus.Remove(unitscene)
					if err := endUnit(ctx, bus, unitid, isEndCloud); err != nil {
						log.Println(err)
					}
				}
			}
		}
	}
}

func main() {
	// Set log format
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 加载配置文件
	configFilePath := flag.String("config_file_path", "", "The config file path.(Required)")
	flag.Parse()
	if *configFilePath == "" {
		log.Fatal("config_file_path is required.")
	}
	config.Load(*configFilePath)

	defer func() {
		log.Println("main goroutine is exit!")
	}()

	// 监听系统信号
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	redconf := config.Config.Redis
	address := redconf.Host + ":" + strconv.Itoa(redconf.Port)
	redisPool := &redis.Pool{
		MaxIdle:     1,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", address)
			if err != nil {
				return nil, err
			}
			if redconf.Auth != "" {
				if _, err := c.Do("AUTH", redconf.Auth); err != nil {
					c.Close()
					return nil, err
				}
			}
			if _, err := c.Do("SELECT", redconf.DB); err != nil {
				c.Close()
				return nil, err
			}
			return c, nil
		},
	}
	defer redisPool.Close()
	ctx := context.WithValue(context.Background(), "redisPool", redisPool)

	// 初始化监测中心
	bus := NewBus()

	// 监控单元是否已结束
	go monitor(ctx, bus)

	ticker := time.NewTicker(scanPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-bus.done:
			log.Println("Bye!")
			return
		case <-ticker.C:
			keys, err := scan(ctx)
			if err != nil {
				log.Println(err)

				// 结束worker
				close(bus.kill)
				select {
				case <-bus.done:
				case <-time.After(time.Second):
					log.Println("Bye bye!")
					return
				}
				return
			}

			log.Println("debug....", keys)

			for _, key := range keys {
				unitid := strings.Replace(key, scanPrefix, "", -1)
				if !bus.In(unitid) {
					bus.Add(unitid)
				}
			}
		case <-interrupt:
			// 结束worker
			close(bus.kill)
			select {
			case <-bus.done:
			case <-time.After(10 * time.Second):
				log.Println("Bye bye!")
				return
			}
			return
		}
	}
}
