package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/darling-kefan/xj/config"
	"github.com/darling-kefan/xj/ndscloud"
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

	router := gin.Default()

	// /v2/units/:unit_id/users?token=:access_token
	// /v2/units/:unit_id/modules/status?token=:access_token
	// /v2/units/:unit_id/modules/list?token=:access_token
	// /v2/units/:unit_id/chat/message?token=:token&chat_id=:id&limit=:limit
	// /v2/ngx/center/units/:unit_id/?token=:access_token

	hub := ndscloud.NewHub()
	go hub.Run()

	v2 := router.Group("/v2")
	{
		//v2.GET("units/:unit_id/users1", ndscloud.ServeUsers)
		v2.GET("units/:unit_id/users", func(c *gin.Context) {
			ndscloud.ServeUsers(hub, c)
		})
		v2.GET("ngx/center/units/:unit_id/", func(c *gin.Context) {
			//ndscloud.ServeWs(hub, c.Writer, c.Request)
			ndscloud.ServeWs(hub, c)
		})
	}

	s := &http.Server{
		Addr:           ":8081",
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	s.ListenAndServe()
}
