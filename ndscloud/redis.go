package ndscloud

import (
	"strconv"

	"github.com/darling-kefan/xj/config"
	"github.com/gomodule/redigo/redis"
)

var (
	// 存储最新场景id(kv)
	// fmt.Sprintf(this, unitId)
	sceneIdKeyFormat string = "nc:unit:scene:id:%s"
	// 存储单元场景信息(kv)
	// fmt.Sprintf(this, unitId, sceneId)
	sceneKeyFormat string = "nc:unit:scene:%s:%d"

	// 存储模块状态指令(hash: instr_mod -> content)
	// fmt.Sprintf(this, unitId, sceneId)
	modInsKeyFormat string = "nc:ins:mod:%s:%d"
	// 模块状态指令历史前缀
	// fmt.Sprintf(this, unitId, sceneId, curmod)
	modInsHistoryKeyFormat string = "nc:ins:mod:his:%s:%d:%s"

	// 群聊(文字聊天)(list)
	// fmt.Sprintf(this, unitId, sceneId)
	chatKeyFormat string = "nc:chat:his:%s:%d"
)

// 创建Redis连接
func connectRedis() (redis.Conn, error) {
	redconf := config.Config.Redis
	address := redconf.Host + ":" + strconv.Itoa(redconf.Port)
	conn, err := redis.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	if redconf.Auth != "" {
		if _, err := conn.Do("AUTH", redconf.Auth); err != nil {
			conn.Close()
			return nil, err
		}
	}
	if _, err := conn.Do("SELECT", redconf.DB); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}
