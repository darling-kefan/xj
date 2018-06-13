package nstat

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/darling-kefan/xj/config"
	_ "github.com/go-sql-driver/mysql"
	"github.com/ipipdotnet/datx-go"
)

// stopCh is an additional signal channel.
// Its sender is moderator goroutine, and its
// receivers are all other goroutines.
var stopCh chan struct{} = make(chan struct{})

// The channel toStop is used to notify the moderator
// to close the additional signal channel (stopCh).
// Its sender is any goroutine, its receivers is the
// moderator goroutine.
var toStop chan string = make(chan string)

// City ip db object
var cityipdb *datx.City

// 加载本地地区库
type Districts struct {
	Country  map[string]int `json:"country"`
	Province map[string]int `json:"province"`
	City     map[string]int `json:"city"`
}

var districtDb *Districts

func NewDistrictDb(disfile string) (*Districts, error) {
	buf, err := ioutil.ReadFile(disfile)
	if err != nil {
		return nil, err
	}
	dises := new(Districts)
	if err = json.Unmarshal(buf, dises); err != nil {
		return nil, err
	}
	return dises, nil
}

// 缓存
type Cache struct {
	sync.RWMutex
	Teachers map[string]bool
}

func NewCache() *Cache {
	cache := &Cache{
		Teachers: make(map[string]bool),
	}
	cache.Reload()
	return cache
}

func (c *Cache) Reload() {
	apiConf := config.Config.MySQL["api"]
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		apiConf.Username,
		apiConf.Password,
		apiConf.Host,
		apiConf.Port,
		apiConf.Dbname,
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		panic(err)
	}

	sql := "SELECT DISTINCT(`uid`) AS `uid` FROM `course_users` WHERE `identity` = 1 AND `deleted_at` IS NULL"
	rows, err := db.Query(sql)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var uid string
	teachers := make(map[string]bool)
	for rows.Next() {
		err := rows.Scan(&uid)
		if err != nil {
			panic(err)
		}
		teachers[uid] = true
	}
	if err = rows.Err(); err != nil {
		panic(err)
	}

	log.Printf("[Cache Reloader] %v\n", teachers)

	c.Lock()
	c.Teachers = teachers
	c.Unlock()
}

func (c *Cache) IsTeacher(uid string) bool {
	c.RLock()
	defer c.RUnlock()
	_, ok := c.Teachers[uid]
	return ok
}

// 全局缓存变量
var cache *Cache

func Run() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("Lauch...........")

	// 加载配置文件
	configFilePath := flag.String("config_file_path", "", "The config file path.(Required)")
	flag.Parse()
	if *configFilePath == "" {
		log.Fatal("config_file_path is required.")
	}
	config.Load(*configFilePath)

	// 加载缓存数据
	cache = NewCache()

	// 加载本地地区库
	var err error
	districtDb, err = NewDistrictDb(config.Config.Stat.Districtdb)
	if err != nil {
		log.Println("Failed to load districtDb: ", err)
		return
	}

	// 创建ip数据定位库对象
	cityipdb, err = datx.NewCity(config.Config.Stat.Cityipdb)
	if err != nil {
		log.Println("Failed to create cityipdb: ", err)
		return
	}

	// 监听系统信号
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	var wg sync.WaitGroup
	wg.Add(4)

	// Start the processor.
	processor := newProcessor()
	go processor.run(&wg)

	// Start the collector.
	collector := newCollector(processor)
	go collector.run(&wg)

	// Start the consumer.
	//for i := 0; i < 5; i++ {
	//	consumer := newConsumer(processor)
	//	go consumer.run(&wg)
	//}
	consumer := newConsumer(processor)
	go consumer.run(&wg)

	// Start Timing Loader(定时更新缓存, 比如定时载入哪些人是老师)
	go func(wg *sync.WaitGroup) {
		log.Printf("Timing loader start...\n")
		defer wg.Done()
		// 启动定时器，每5分钟更新一次缓存
		ticker := time.NewTicker(300 * time.Second)
		defer ticker.Stop()
	loop:
		for {
			select {
			case <-ticker.C:
				cache.Reload()
			case <-stopCh:
				break loop
			}
		}
		log.Printf("Timing loader quit...\n")
	}(&wg)

	// Start the moderator goroutine.
	go func() {
		select {
		case <-toStop:
			close(stopCh)
		case <-interrupt:
			close(stopCh)
		}
		log.Println("Moderator goroutine quit...")
	}()

	wg.Wait()
	log.Println("Main goroutine quit...")
}
