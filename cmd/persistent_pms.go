// 持久化笔迹流到文件
// TODO 改成守护进程多协程的方式保存笔迹流
//
// go run persistent_pms.go del --unit_id=A16 --scene_id=3 --uid=123 // 删除笔迹流
// go run persistent_pms.go save // 保存笔迹流
package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	//"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/darling-kefan/xj/config"
	"github.com/darling-kefan/xj/helper"
	"github.com/gomodule/redigo/redis"
)

func ctxKey(key string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(key)))
}

func main() {
	// Set log format
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Subcommands
	saveCmd := flag.NewFlagSet("save", flag.ExitOnError)
	delCmd := flag.NewFlagSet("del", flag.ExitOnError)

	// Del subcommand flag pointers
	unitidPtr := delCmd.String("unit_id", "", "the unit id.(Required)")
	sceneidPtr := delCmd.Int("scene_id", 0, "the scene id of unit.(Required)")
	uidPtr := delCmd.Int("uid", 0, "the user id.(Required)")

	// Verify that a subcommand has been provided
	if len(os.Args) < 2 {
		fmt.Println("save or del subcommand is required")
		os.Exit(1)
	}

	// Switch on the subcommand
	// Parse the flags for appropriate FlagSet
	switch os.Args[1] {
	case "save":
		saveCmd.Parse(os.Args[2:])
	case "del":
		delCmd.Parse(os.Args[2:])
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

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

	ctx := context.WithValue(context.Background(), ctxKey("redisPool"), redisPool)
	usmap, err := getUnitScenes(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if saveCmd.Parsed() {
		var wg sync.WaitGroup
		// 检查单元是否在进行中
		for unitid, sceneid := range usmap {
			// Increment the WaitGroup counter.
			wg.Add(1)
			go run(ctx, &wg, unitid, sceneid)

			// 匿名函数一定要接收unitid, sceneid参数；否则函数体中的unitid, sceneid可以在函数体外的被更改。
			//go func(unitid string, sceneid int) {
			//	// Decrement the counter when the goroutine completes.
			//	defer wg.Done()
			//
			//	running, err := checkUnit(ctx, unitid)
			//	if err != nil {
			//		log.Println(err)
			//		return
			//	}
			//	log.Println(unitid, sceneid, running, err)
			//	if running {
			//		// 写入二进制文件
			//		if err := writePms(ctx, unitid, sceneid); err != nil {
			//			log.Println(err)
			//			return
			//		}
			//	}
			//}(unitid, sceneid)
		}

		// Wait for all goroutines to complete.
		wg.Wait()
	}

	if delCmd.Parsed() {
		if err := deletepms(ctx, *unitidPtr, *sceneidPtr, *uidPtr); err != nil {
			log.Fatal(err)
		}
	}
}

// 主程序
// 注意：此处wg一定是引用传递*sync.WaitGroup，否则进行不能正常结束。
func run(ctx context.Context, wg *sync.WaitGroup, unitid string, sceneid int) {
	// Decrement the counter when the goroutine completes.
	defer wg.Done()

	running, err := checkUnit(ctx, unitid)
	if err != nil {
		log.Printf("%s, break\n", err)
		return
	}
	log.Println(unitid, sceneid, running, err)
	if running {
		// 写入二进制文件
		if err := writePms(ctx, unitid, sceneid); err != nil {
			log.Println(err)
			return
		}
	}
}

// 持久化笔迹流
func writePms(ctx context.Context, unitid string, sceneid int) error {
	redisPool := ctx.Value(ctxKey("redisPool")).(*redis.Pool)
	red := redisPool.Get()
	defer red.Close()

	// 查找所有用户的笔迹流
	match := fmt.Sprintf("nc:pms:%s:%d:*", unitid, sceneid)
	var iter int
	var keys []string
	for {
		if bulks, err := redis.MultiBulk(red.Do("SCAN", iter, "MATCH", match, "COUNT", 100)); err != nil {
			return err
		} else {
			iter, _ = redis.Int(bulks[0], nil)
			subkeys, _ := redis.Strings(bulks[1], nil)
			keys = append(keys, subkeys...)
		}
		if iter == 0 {
			break
		}
	}

	// 遍历所有笔迹流用户，并判断其身份
	// teapmskey用于存放老师，供笔迹流回看接口使用
	teapmskey := fmt.Sprintf("nc:pms:teacher:%s:%d", unitid, sceneid)
	for _, pmskey := range keys {
		uid, err := strconv.Atoi(strings.Split(pmskey, ":")[4])
		if err != nil {
			return err
		}

		ci, err := courseidt(ctx, unitid, uid)
		if err != nil {
			log.Println(err)
			continue
		}
		if ci != nil {
			if ci.Identity == 1 {
				if _, err := red.Do("ZADD", teapmskey, time.Now().Unix(), uid); err != nil {
					return err
				}
				log.Printf("ZADD %s %d %d\n", teapmskey, time.Now().Unix(), uid)
			}
		}
	}

	// 从redis读取笔迹流并写入到文件
	var buf bytes.Buffer
	for _, pmskey := range keys {
		// 笔迹流文件路径
		pf := pmsfile(pmskey)
		fp, err := os.OpenFile(pf, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer fp.Close()

		// 获取初始读取位置
		startkey := strings.Replace(pmskey, "pms", "pms:offset", -1)
		start := 0
		if startraw, err := red.Do("GET", startkey); err != nil {
			return err
		} else {
			if startraw != nil {
				start, err = redis.Int(startraw, nil)
				if err != nil {
					return err
				}
			}
		}
		// 分页读取笔迹流
		for i := 1; i < 1<<10; i++ {
			length := 100
			start = start + (i-1)*length
			end := start + length - 1
			pmses, err := redis.ByteSlices(red.Do("LRANGE", pmskey, start, end))
			if err != nil {
				return err
			}
			log.Printf("LRANGE %s %d %d\n", pmskey, start, end)
			for _, pms := range pmses {
				buf.Write(pms)
			}
			if _, err := buf.WriteTo(fp); err != nil {
				return err
			}
			buf.Reset()

			if len(pmses) < length {
				start = start + len(pmses)
				break
			}
		}
		if _, err := red.Do("SET", startkey, start); err != nil {
			return err
		}
		log.Printf("SET %s %d\n", startkey, start)
	}

	return nil
}

// 删除笔迹流文件
func deletepms(ctx context.Context, unitid string, sceneid int, uid int) error {
	redisPool := ctx.Value(ctxKey("redisPool")).(*redis.Pool)
	red := redisPool.Get()
	defer red.Close()

	pmskey := fmt.Sprintf("nc:pms:%s:%d:%d", unitid, sceneid, uid)
	pmsoffkey := strings.Replace(pmskey, "pms", "pms:offset", -1)
	if _, err := red.Do("DEL", pmsoffkey); err != nil {
		return err
	}
	if err := os.Remove(pmsfile(pmskey)); err != nil {
		return err
	}
	return nil
}

// 笔迹流存放路径
func pmsfile(pmskey string) string {
	pmsparts := strings.Split(pmskey, ":")
	filename := fmt.Sprintf("pms_%s_%s_%s.binary", pmsparts[2], pmsparts[3], pmsparts[4])
	return path.Join(config.Config.Common.Pmspath, filename)
}

type CourseIdentity struct {
	Uid      int
	Identity int
	CourseId int
}

// 查询单元身份
func courseidt(ctx context.Context, unitid string, uid int) (*CourseIdentity, error) {
	// OAuth2认证
	redisPool := ctx.Value(ctxKey("redisPool")).(*redis.Pool)
	token, err := helper.AccessToken(redisPool.Get(), "client_credentials", nil)
	if err != nil {
		return nil, err
	}

	var api string
	api = strings.Replace(config.Config.Api.UnitIdentity, ":unit_id", unitid, -1)
	api = strings.Replace(api, ":uid", strconv.Itoa(uid), -1)

	api = fmt.Sprintf("%s?token=%s", api, token)
	resp, err := http.Get(api)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	type Response struct {
		Errcode int64          `json:"errcode"`
		Errmsg  string         `json:"errmsg"`
		Data    CourseIdentity `json:"data"`
	}
	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	if response.Errcode != 0 {
		return nil, errors.New(response.Errmsg)
	}
	return &response.Data, nil
}

// 检查单元是否在进行中
func checkUnit(ctx context.Context, unitid string) (bool, error) {
	// OAuth2认证
	redisPool := ctx.Value(ctxKey("redisPool")).(*redis.Pool)
	token, err := helper.AccessToken(redisPool.Get(), "client_credentials", nil)
	if err != nil {
		return false, err
	}

	// 查询单元接口
	unitApi := strings.Replace(config.Config.Api.UnitInfo, ":unit_id", unitid, -1)
	unitApi = fmt.Sprintf("%s?token=%s", unitApi, token)
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
		return false, errors.New("The unit: " + unitid + " is not exists")
	}
	return false, errors.New("The unit: " + unitid + " is not running")
}

// 获取所有单元场景，判断单元是否进行中；如果进行中，从笔迹流队列拉取消息写入到二进制文件
func getUnitScenes(ctx context.Context) (map[string]int, error) {
	// redis连接
	redisPool := ctx.Value(ctxKey("redisPool")).(*redis.Pool)
	conn := redisPool.Get()
	defer conn.Close()

	// 获取所有单元ID
	match := "nc:ins:mod:*"
	iter := 0
	var units []string
	for {
		if bulks, err := redis.MultiBulk(conn.Do("SCAN", iter, "MATCH", match, "COUNT", 1000)); err != nil {
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
		// iter == 0标识迭代结束
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

	// 获取单元的最新场景
	// redis pipelines批量执行redis命令
	for _, unitid := range unitlist {
		sceneidkey := "nc:unit:scene:id:" + unitid
		conn.Send("GET", sceneidkey)
	}
	conn.Flush()
	for _, unitid := range unitlist {
		res, _ := conn.Receive()
		// 如果不存在最新场景，则默认为1
		if res == nil {
			usmap[unitid] = 1
		} else {
			if sceneid, err := redis.Int(res, nil); err == nil {
				usmap[unitid] = sceneid
			} else {
				return nil, err
			}
		}
	}
	return usmap, nil
}
