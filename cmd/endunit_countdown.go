package main

// 1. Scan Redis KEY="nc:chan:unit:run:{unitid}", 定时执行Scan, 并维护一个监控表
//
// 2. Start a goroutine to monitor unit
// > 监测KEY是否存在，如果不存在则表示单元结束，通知云中控结束单元，通知Canvas结束单元；
// > 将该单元加入到监控表, 单元结束后从监控表移除；
//
// 注意点：
// > C-c结束进程，需终止所有goroutine

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/darling-kefan/xj/config"
	"github.com/darling-kefan/xj/helper"
)

const (
	scanPrefix    string        = "nc:chan:unit:run:"
	scanPeriod    time.Duration = 1 * time.Second
	monitorPeriod time.Duration = 1 * time.Second
)

// 管理所有监测中的单元
type Bus struct {
	units map[string]bool
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
		units: make(map[string]bool),
		// 注意：此处一定要初始化
		mutex: new(sync.RWMutex),
		done:  make(chan struct{}),
		kill:  make(chan struct{}),
	}
}

// 判断该unitid是否在监测中
func (b *Bus) In(unitid string) bool {
	b.mutex.RLock()
	_, isIn := b.units[unitid]
	b.mutex.RUnlock()
	return isIn
}

// 将单元添加到监控中
func (b *Bus) Add(unitid string) {
	b.mutex.Lock()
	b.units[unitid] = true
	b.mutex.Unlock()
}

// 将单元从监控中移除
func (b *Bus) Remove(unitid string) {
	b.mutex.Lock()
	delete(b.units, unitid)
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
func endUnit(ctx context.Context, bus *Bus, unitid string) error {
	// redis实例
	redconn := ctx.Value("redisPool").(*redis.Pool).Get()

	// 在监控中心移除该单元
	bus.Remove(unitid)

	// 通知云中控单元结束
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
	subchan := "nc:chan:all:" + unitid
	if _, err = redconn.Do("PUBLISH", subchan, string(b)); err != nil {
		return err
	}
	log.Println("publish", subchan, string(b))

	// 通知Canvas单元结束
	token, err := helper.AccessToken(redconn, "client_credentials", nil)
	if err != nil {
		return err
	}
	api := config.Config.Api.Domain + "/v1/units/:unit_id/status?token=:token"
	api = strings.Replace(strings.Replace(api, ":unit_id", unitid, -1), ":token", token, -1)
	resp, err := http.Get(api)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	log.Println(api, string(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	type Response struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}
	if response.Errcode != 0 {
		return errors.New(response.Errmsg)
	}
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
			log.Println("monitor goroutine, debug..........", bus.units)
			for unitid, _ := range bus.units {
				key := scanPrefix + unitid
				isExists, err := redis.Bool(redconn.Do("EXISTS", key))
				if err != nil {
					log.Println(err)
					return
				}
				// TODO test suit
				//if err := endUnit(ctx, bus, unitid); err != nil {
				//	log.Println(err)
				//}
				if !isExists {
					if err := endUnit(ctx, bus, unitid); err != nil {
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
