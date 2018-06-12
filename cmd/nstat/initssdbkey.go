package main

import (
	"flag"
	"log"

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
	val, err := db.Get("nstat:1:1:20180601")
	log.Println(val, err)
}
