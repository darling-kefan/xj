package main

import (
	"encoding/json"
	"fmt"
	"log"
	//"sync"
	"strconv"
	"strings"
	"time"
	//"github.com/darling-kefan/xj/ndscloud"

	"github.com/gomodule/redigo/redis"

	"github.com/darling-kefan/xj/config"
)

type User struct {
	Name string
}

func list(mx map[string]*User) []User {
	l := make([]User, 0)
	for _, item := range mx {
		l = append(l, *item)
	}
	return l
}

// Parse the field to of message and return groups and individuals
func ParseFieldTo(to string) (groups, individuals []string) {
	if to == "" {
		return
	}
	parts := strings.Split(to, "@")
	switch {
	case parts[0] == "A":
		groups = []string{"A"}
		return
	case parts[0] == "T":
		groups = []string{"T"}
	case parts[0] == "S":
		groups = []string{"S"}
	case parts[0] == "D":
		groups = []string{"D"}
	case strings.Index(parts[0], "|") != -1:
		groups := strings.Split(parts[0], "|")
		hasA := false
		for _, v := range groups {
			if v == "A" {
				hasA = true
				break
			}
		}
		if hasA {
			groups = []string{"A"}
			return
		}
	case strings.Index(parts[0], ",") != -1:
		individuals = strings.Split(parts[0], ",")
		return
	default:
		individuals = []string{parts[0]}
		return
	}
	if parts[1] != "" {
		ia := strings.Split(parts[1], ",")
		individuals = append(individuals, ia...)
	}
	return
}

func main() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	groups, individuals := ParseFieldTo("A|S@1,2")
	log.Println(groups, individuals)

	return

	// 加载配置文件
	configFilePath := "/home/shouqiang/go/src/github.com/darling-kefan/xj"
	config.Load(configFilePath)

	redconf := config.Config.Redis
	address := redconf.Host + ":" + strconv.Itoa(redconf.Port)
	redisPool := &redis.Pool{
		MaxIdle:     1,
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
	defer redisPool.Close()

	redconn := redisPool.Get()

	unitid := "A16"
	sceneid := "21"

	// 判断云中控是否已经结束课程
	isEndCloud := false
	scenekey := fmt.Sprintf("nc:unit:scene:%s:%s", unitid, sceneid)
	scenebytes, _ := redis.Bytes(redconn.Do("GET", scenekey))
	if len(scenebytes) > 0 {
		var sceneinfo *struct {
			Endtime float64 `json:"end_time"`
		} = &struct {
			Endtime float64 `json:"end_time"`
		}{}
		err := json.Unmarshal(scenebytes, sceneinfo)
		log.Printf("%#v\n", sceneinfo)
		if err != nil {
			log.Println(err)
		}
		if sceneinfo.Endtime > 0 {
			isEndCloud = true
		}
	}
	log.Printf("%#v\n", isEndCloud)

	ttl, err := redis.Int(redconn.Do("TTL", "test:ttl"))
	log.Printf("%#v %#v\n", ttl, err)

	//mx := make(map[string]*User)
	//user := &User{Name: "shouqiang"}
	//mx["A"] = user
	//
	//for k, v := range mx {
	//	log.Println(k, v.Name)
	//}
	//
	//// 值复制，修改l不会影响mx
	//l := list(mx)
	//l[0].Name = "kefan"
	//log.Println(l)
	//
	//for k, v := range mx {
	//	log.Println(k, v.Name)
	//}

	//sss := make([]int, 0)
	//go func(n int) {
	//	sss = append(sss, n)
	//}(1)

	//log.Printf("%d\n", sss[0])

	//var xxx int
	//go func() {
	//	xxx = xxx + 1
	//}()
	//log.Printf("%#v\n", xxx)
	//
	//return
	//
	//var x interface{}
	//
	//y := 1
	//x = y
	//
	//switch n := x.(type) {
	//case int:
	//	log.Println("I am int", n)
	//default:
	//}
	//
	//return
	//
	//var interf interface{}
	//log.Printf("%#v\n", interf == nil)
	//return
	//
	//// 空结构体赋值方法
	//
	//// 方法一: 声明struct{}
	//var s struct{}
	//log.Printf("%#v\n", s)
	//// Output:
	//// 2018/05/23 15:10:29 test.go:19: struct {}{}
	//
	//// s = nil // 错误，nil不能赋给结构体，nil不是结构体类型的空值
	//
	//// 方法二：使用struct{}{}赋值空结构体变量
	//var set map[string]struct{}
	//set = make(map[string]struct{})
	//
	//set["a"] = struct{}{}
	//set["v"] = struct{}{}
	//
	//log.Printf("%#v\n", set)
	//// Output:
	//// 2018/05/23 15:10:29 test.go:30: map[string]struct {}{"v":struct {}{}, "a":struct {}{}
	//
	//return
	//
	//msg, err := ndscloud.UnmarshalMessage([]byte(`{"act1": "1", "os": "1", "vi": "1", "hw": "0"}`))
	//log.Printf("%#v %#v\n", msg, err)
	//if err == nil {
	//	log.Printf("%#v\n", msg.(*ndscloud.RegMsg).Dt)
	//}
	//
	//msg, err = ndscloud.UnmarshalMessage([]byte(`{"act": "3", "usr": [{"uid": "2", "nm": "小新", "sex": "1", "idt": "1", "os": "1", "vi": "1", "hw": "1"}, {"uid": "3", "nm": "小白", "sex": "1", "idt": "2", "os": "1", "vi": "1", "hw": "1"}]}`))
	//log.Printf("%#v\n", msg)
	//
	//msg, err = ndscloud.UnmarshalMessage([]byte(`{"act": "6", "from": "101:小新", "to": "D|S|@76,101", "msg": {"key": "value"}}`))
	//log.Printf("%#v %#v\n", msg, err)
	//
	//b, err := json.Marshal(msg)
	//log.Printf("%#v\n", string(b))
	//
	//return
	//
	//var jsonRaw = []byte(`{"created_at": "2018-03-05T12:30:00Z"}`)
	//
	//type Response struct {
	//	CreatedAt time.Time `json:"created_at"`
	//}
	//var resp Response
	//
	//err = json.Unmarshal(jsonRaw, &resp)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//log.Println(resp)
}
