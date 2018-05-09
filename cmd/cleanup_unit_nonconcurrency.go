// Scan Redis中所有单元id，判断单元是否在进行中；如果不在进行中，则清理该单元下的所有在线用户记录
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/darling-kefan/xj/config"
	"github.com/darling-kefan/xj/helper"
	"github.com/gomodule/redigo/redis"
)

type Env struct {
	Redis redis.Conn
}

var env *Env = new(Env)

func init() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	redconf := config.Config.Redis
	address := redconf.Host + ":" + strconv.Itoa(redconf.Port)
	red, err := redis.Dial("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	if redconf.Auth != "" {
		if _, err := red.Do("AUTH", redconf.Auth); err != nil {
			red.Close()
			log.Fatal(err)
		}
	}
	if _, err := red.Do("SELECT", redconf.DB); err != nil {
		red.Close()
		log.Fatal(err)
	}
	env.Redis = red
}

// 获取所有单元
func (env *Env) getUnits() ([]string, error) {
	match := "nc:ins:mod:*"
	iter := 0
	var units []string
	for {
		if bulks, err := redis.MultiBulk(env.Redis.Do("SCAN", 0, "MATCH", match, "COUNT", 100)); err != nil {
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
func (env *Env) checkUnit(unitid string) (bool, error) {
	// OAuth2认证
	token, err := helper.AccessToken(env.Redis, "client_credentials", nil)
	if err != nil {
		return false, err
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
		return false, err
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
		return false, err
	}
	if data.Errcode == 0 && data.InnerData.Status == "1" {
		return true, nil
	}
	if data.Errcode == 12302 {
		return false, errors.New("The unit: " + unitid + " is not exists.")
	}
	return false, errors.New("The unit: " + unitid + " is not running.")
}

// 清理掉在线用户
func (env *Env) clearupOnlines(unitid string) error {
	onlinekey := "nc:onlines:" + unitid
	ok, err := redis.Bool(env.Redis.Do("del", onlinekey))
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
		if bulks, err := redis.MultiBulk(env.Redis.Do("SCAN", iter, "MATCH", match, "COUNT", 100)); err != nil {
			return err
		} else {
			iter, _ = redis.Int(bulks[0], nil)
			keys, _ = redis.Strings(bulks[1], nil)
			for _, key := range keys {
				ok, err = redis.Bool(env.Redis.Do("DEL", key))
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

	// 2. 判断单元是否进行中，如果否则清理在线终端列表
	for _, unitid := range units {
		if isRunning, err := env.checkUnit(unitid); err != nil {
			if err := env.clearupOnlines(unitid); err != nil {
				log.Fatal(err)
			}
		} else {
			if !isRunning {
				if err := env.clearupOnlines(unitid); err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}
