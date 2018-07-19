package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/darling-kefan/xj/config"
	"github.com/darling-kefan/xj/ssdb"
)

func main() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 加载配置文件
	configFilePath := flag.String("config_file_path", "", "The config file path.(Required)")
	flag.Parse()
	if *configFilePath == "" {
		log.Fatal("config_file_path is required.")
	}
	config.Load(*configFilePath)

	host := config.Config.SSDB.Host
	port := config.Config.SSDB.Port
	auth := config.Config.SSDB.Auth
	db, err := ssdb.Connect(host, port)
	if err != nil {
		log.Fatal(err)
	}
	if auth != "" {
		res, err := db.Do("auth", auth)
		if res[0] != "ok" {
			log.Fatal(res, err)
		}
	}

	var resp, kvs, kps []string
	var scanStart, scanEnd string
	var todayKey string

	yesterday := time.Now().AddDate(0, 0, -1).Format("20060102")
	today := time.Now().Format("20060102")
	scanLimit := 1000
	// 待初始化的stype值(用户总数，课程总数，课件总数，占用空间)
	var stypes []int = []int{1, 21, 41, 42, 51, 52}
	for _, stype := range stypes {
		scanStart = fmt.Sprintf("nstat:%d:", stype)
		scanEnd = fmt.Sprintf("nstat:%d:z", stype)
		for {
			resp, err = db.Do("scan", scanStart, scanEnd, scanLimit)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("scan %s %s %d : %v\n", scanStart, scanEnd, scanLimit, resp)
			kvs = resp[1:]
			if len(kvs) == 0 {
				break
			}
			for i := 0; i < len(kvs); i = i + 2 {
				kps = strings.Split(kvs[i], ":")
				if kps[3] == yesterday {
					todayKey = strings.Replace(kvs[i], yesterday, today, -1)
					resp, err = db.Do("incr", todayKey, kvs[i+1])
					if err != nil {
						log.Fatal(err)
					}
					if len(resp[1:]) > 0 {
						log.Printf("incr %s %s : %v\n", todayKey, kvs[i+1], resp)
					}
				}
			}
			scanStart = kvs[len(kvs)-2]
		}
	}
}
