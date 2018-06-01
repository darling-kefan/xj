package helper

import (
	"log"
	"os"
	"path/filepath"
)

// 获取程序入口文件的绝对路径
func CurrPath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return dir
}

func inSlice(element interface{}, array []interface{}) bool {
	return false
}
