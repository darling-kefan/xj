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
		did int
		pid int
	)
	cps := make(map[int]int) // city_did => province_did
	sql := "SELECT `did`, `pid` from `districts` WHERE `leveltype` = 2"
	rows, err := db.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&did, &pid)
		if err != nil {
			log.Fatal(err)
		}
		cps[did] = pid
	}
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}

	filename := "cps.txt"
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

	content, err := json.Marshal(cps)
	if err != nil {
		panic(err)
	}
	if _, err := f.Write(content); err != nil {
		panic(err)
	}
}
