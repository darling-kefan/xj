package main

import (
	"flag"
	"log"

	"github.com/darling-kefan/xj/config"
	"github.com/darling-kefan/xj/nstat"
	"github.com/darling-kefan/xj/nstat/protocol"
)

func main() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("Main start...")

	// 加载配置文件
	configFilePath := flag.String("config_file_path", "", "The config file path.(Required)")
	flag.Parse()
	if *configFilePath == "" {
		log.Fatal("config_file_path is required.")
	}
	config.Load(*configFilePath)

	factors := []*protocol.StatFactor{
		&protocol.StatFactor{
			Stype:  "23",
			Oid:    "1",
			Sid:    "100",
			Subkey: "A",
			Value:  10,
		},
		&protocol.StatFactor{
			Stype:  "23",
			Oid:    "1",
			Sid:    "100",
			Subkey: "B",
			Value:  15,
		},
		&protocol.StatFactor{
			Stype:  "23",
			Oid:    "1",
			Sid:    "100",
			Subkey: "C",
			Value:  8,
		},
		&protocol.StatFactor{
			Stype:  "23",
			Oid:    "1",
			Sid:    "100",
			Subkey: "D",
			Value:  5,
		},
		&protocol.StatFactor{
			Stype:  "23",
			Oid:    "1",
			Sid:    "100",
			Subkey: "E",
			Value:  1,
		},
	}

	log.Println(factors)

	// 请求接口，同步到ssdb
	attempt := 1
	for {
		if err := nstat.CommitFactors(factors); err != nil {
			log.Printf("Failed to commit factors, error: %s\n", err)
			attempt = attempt + 1
		} else {
			break
		}
		if attempt > 3 {
			break
		}
	}
}
