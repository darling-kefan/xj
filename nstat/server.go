package nstat

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/darling-kefan/xj/config"
	"github.com/ipipdotnet/datx-go"
)

// City ip db object
var cityipdb *datx.City

// The map of name and did
var countryMap, provinceMap, cityMap map[string]string

// stopCh is an additional signal channel.
// Its sender is moderator goroutine, and its
// receivers are all other goroutines.
var stopCh chan struct{} = make(chan struct{})

// The channel toStop is used to notify the moderator
// to close the additional signal channel (stopCh).
// Its sender is any goroutine, its receivers is the
// moderator goroutine.
var toStop chan string = make(chan string)

// 存储最新消费日志的位置
var Los *LogOffsetSet = NewLogOffsetSet()

type LogOffsetSet struct {
	sync.RWMutex
	offsets map[string]string
}

func NewLogOffsetSet() *LogOffsetSet {
	return &LogOffsetSet{
		offsets: make(map[string]string),
	}
}

func (los *LogOffsetSet) Set(topic string, msgid string) {
	los.Lock()
	los.offsets[topic] = msgid
	los.Unlock()
}

func (los *LogOffsetSet) Map() map[string]string {
	los.RLock()
	mapset := make(map[string]string)
	for topic, msgid := range los.offsets {
		mapset[topic] = msgid
	}
	los.RUnlock()
	return mapset
}

func (los *LogOffsetSet) Clear() {
	los.Lock()
	los.offsets = make(map[string]string)
	los.Unlock()
}

// 存储消费失败的消息
var Lfs *LogFailedSet = NewLogFailedSet()

type LogFailedSet struct {
	sync.RWMutex
	idmap map[string]map[string]struct{}
}

func NewLogFailedSet() *LogFailedSet {
	return &LogFailedSet{
		idmap: make(map[string]map[string]struct{}),
	}
}

func (lfs *LogFailedSet) Add(topic string, msgid string) {
	lfs.Lock()
	if _, ok := lfs.idmap[topic]; ok {
		lfs.idmap[topic][msgid] = struct{}{}
	} else {
		lfs.idmap[topic] = make(map[string]struct{})
	}
	lfs.Unlock()
}

func (lfs *LogFailedSet) Map() map[string]map[string]struct{} {
	lfs.RLock()
	mapset := make(map[string]map[string]struct{})
	for topic, value := range lfs.idmap {
		mapset[topic] = make(map[string]struct{})
		for msgid, _ := range value {
			mapset[topic][msgid] = struct{}{}
		}
	}
	lfs.RUnlock()
	return mapset
}

func (lfs *LogFailedSet) Clear() {
	lfs.Lock()
	lfs.idmap = make(map[string]map[string]struct{})
	lfs.Unlock()
}

// main goroutine退出时将最新message_id持久化

type Districts struct {
	Country  map[string]int `json:"country"`
	Province map[string]int `json:"province"`
	City     map[string]int `json:"city"`
}

var districtDb *Districts

func loadDistricts(disfile string) (*Districts, error) {
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

	// 加载本地地区库
	var err error
	districtDb, err = loadDistricts(config.Config.Stat.Districtdb)
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
	wg.Add(3)

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
