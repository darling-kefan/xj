// Scan Redis中所有单元id，判断单元是否在进行中；如果不在进行中，则清理该单元下的所有在线用户记录
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/darling-kefan/xj/config"
	"github.com/darling-kefan/xj/helper"
	"github.com/gomodule/redigo/redis"
)

type Env struct {
	RedisPool *redis.Pool
}

var env *Env = new(Env)

type RD struct {
	Unitid  string
	Running bool
}

func init() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	redconf := config.Config.Redis
	address := redconf.Host + ":" + strconv.Itoa(redconf.Port)
	redisPool := &redis.Pool{
		MaxIdle:     3,
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
	env.RedisPool = redisPool
}

// 获取所有单元
func (env *Env) getUnits() ([]string, error) {
	red := env.RedisPool.Get()
	defer red.Close()

	match := "nc:ins:mod:*"
	iter := 0
	var units []string
	for {
		if bulks, err := redis.MultiBulk(red.Do("SCAN", 0, "MATCH", match, "COUNT", 100)); err != nil {
			return nil, err
		} else {
			iter, _ = redis.Int(bulks[0], nil)
			keys, _ := redis.Strings(bulks[1], nil)
			for _, key := range keys {
				// 提取unit_id
				subkey := key[11:]
				parts := strings.Split(subkey, ":")
				if parts[0] != "his" && parts[0] != "off" {
					units = append(units, parts[0])
				}
			}
		}
		// iter == 0标志着迭代结束
		if iter == 0 {
			break
		}
	}

	// unitid去重
	usmap := make(map[string]int)
	for _, unitid := range units {
		usmap[unitid] = 0
	}
	// unitlist固定顺序的非重复队列
	i := 0
	unitlist := make([]string, len(usmap))
	for unitid, _ := range usmap {
		unitlist[i] = unitid
		i = i + 1
	}
	log.Println("units: ", unitlist)
	return unitlist, nil
}

// 检查单元是否在进行中
func (env *Env) checkUnit(unitid string, ch chan RD) {
	red := env.RedisPool.Get()
	defer red.Close()

	// 初始化值
	rd := RD{Unitid: unitid}

	// OAuth2认证
	token, err := helper.AccessToken(red, "client_credentials", nil)
	if err != nil {
		log.Println(err)
		ch <- rd
		return
	}

	// 查询单元接口
	unitApi := strings.Replace(config.Config.Api.UnitInfo, ":unit_id", unitid, -1)
	unitApi = fmt.Sprintf("%s?token=%s", unitApi, token)

	//tr := &http.Transport{
	//	TLSClientConfig: &tls.Config{
	//		InsecureSkipVerify: true,
	//	},
	//}
	//client := &http.Client{Transport: tr}
	resp, err := http.Get(unitApi)
	if err != nil {
		log.Println(err)
		ch <- rd
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	type InnerData struct {
		UnitId    string `json:"unit_id"`
		Status    string `json:"status"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}
	type Data struct {
		Errcode   int64     `json:"errcode"`
		Errmsg    string    `json:"errmsg"`
		InnerData InnerData `json:"data"`
	}
	var data Data
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Println(err)
		ch <- rd
		return
	}
	if data.Errcode == 0 && data.InnerData.Status == "1" {
		rd.Running = true
		ch <- rd
		return
	}
	if data.Errcode == 12302 {
		log.Println("The unit: " + unitid + " is not exists.")
		ch <- rd
		return
	}
	log.Println("The unit: " + unitid + " is not running.")
	ch <- rd
}

// 清理掉在线用户
func (env *Env) clearupOnlines(unitid string) error {
	red := env.RedisPool.Get()
	defer red.Close()

	onlinekey := "nc:onlines:" + unitid
	ok, err := redis.Bool(red.Do("del", onlinekey))
	if err != nil {
		return err
	}
	if ok {
		log.Printf("DEL %s\n", onlinekey)
	}

	match := fmt.Sprintf("nc:onlines:lc:%s:*", unitid)
	iter := 0
	keys := make([]string, 0)
	for {
		if bulks, err := redis.MultiBulk(red.Do("SCAN", iter, "MATCH", match, "COUNT", 100)); err != nil {
			return err
		} else {
			iter, _ = redis.Int(bulks[0], nil)
			keys, _ = redis.Strings(bulks[1], nil)
			for _, key := range keys {
				ok, err = redis.Bool(red.Do("DEL", key))
				if err != nil {
					return err
				}
				if ok {
					log.Printf("DEL %s\n", key)
				}
			}
		}
		if iter == 0 {
			break
		}
	}

	return nil
}

func main() {
	// 1. 查找所有单元
	units, err := env.getUnits()
	if err != nil {
		log.Fatal(err)
	}

	runch := make(chan RD, len(units))

	// 2. 判断单元是否进行中，如果否则清理在线终端列表
	for _, unitid := range units {
		go env.checkUnit(unitid, runch)
	}

	var checkedUnits []string
	for data := range runch {
		log.Println(data)
		if data.Running {
			if err := env.clearupOnlines(data.Unitid); err != nil {
				log.Fatal(err)
			}
		}
		checkedUnits = append(checkedUnits, data.Unitid)
		// 从通道接收到值的数量等于启动的goroutine个数，也等于单元个数
		// 如果接手到值的数量等于单元个数，说明所有goroutine都已结束，因此可以在main goroutine关闭通道～
		if len(checkedUnits) == len(units) {
			close(runch)
			break
		}
	}
}
