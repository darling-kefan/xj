// go run district.go --config_file_path '/home/shouqiang/go/src/github.com/darling-kefan/xj'
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/darling-kefan/xj/config"
	_ "github.com/go-sql-driver/mysql"
)

type Districts struct {
	Country  map[string]int `json:"country"`
	Province map[string]int `json:"province"`
	City     map[string]int `json:"city"`
}

func main() {
	// 加载配置文件
	configFilePath := flag.String("config_file_path", "", "The config file path.(Required)")
	flag.Parse()
	if *configFilePath == "" {
		log.Fatal("config_file_path is required.")
	}
	config.Load(*configFilePath)

	accountConf := config.Config.MySQL["account"]
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		accountConf.Username,
		accountConf.Password,
		accountConf.Host,
		accountConf.Port,
		accountConf.Dbname)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	var (
		did       int
		name      string
		shortname string
		leveltype int
	)
	districts := &Districts{
		Country:  make(map[string]int),
		Province: make(map[string]int),
		City:     make(map[string]int),
	}
	sql := "SELECT `did`, `name`, `shortname`, `leveltype` from `districts` WHERE `leveltype` <= 2"
	rows, err := db.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&did, &name, &shortname, &leveltype)
		if err != nil {
			log.Fatal(err)
		}
		switch leveltype {
		case 0:
			districts.Country[name] = did
			districts.Country[shortname] = did
		case 1:
			districts.Province[name] = did
			districts.Province[shortname] = did
		case 2:
			districts.City[name] = did
			districts.City[shortname] = did
		}
	}
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}

	filename := config.Config.Stat.Districtdb
	if _, err := os.Stat(filename); os.IsExist(err) {
		os.Remove(filename)
	}

	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()

	content, err := json.Marshal(districts)
	if err != nil {
		panic(err)
	}
	if _, err := f.Write(content); err != nil {
		panic(err)
	}
}
