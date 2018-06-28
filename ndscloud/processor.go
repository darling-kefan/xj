package ndscloud

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
)

// 负责从客户端接收消息，并解析、处理、转发等
// 主要包括：json消息(文本消息)处理器 和 笔迹流消息处理器
func (c *Client) process(raw []byte) {
	unmarshalRaw, err := UnmarshalMessage(raw)
	if err != nil {
		log.Printf("[%s] %s\n", c.id, err)
		return
	}

	switch message := unmarshalRaw.(type) {
	case *RegMsg:
		// 判断注册消息是否和客户端身份匹配
		if (c.isDevice() && message.Dt == "") || (c.isUser() && message.Dt != "") {
			log.Printf("[%s] bad registration message format: user connect!\n", c.id)
			return
		}

		// 判断是否广播上线消息，一个客户端上线只广播一次消息
		isSendOnlineMsg := false

		// 注册时间只在第一次注册时设置
		if !c.isRegistered {
			switch v := c.info.(type) {
			case *UserInfo:
				v.Vi = message.Vi
				v.Hw = message.Hw
				if message.Os != "" {
					v.Os = message.Os
				}
			case *DeviceInfo:
				v.Vi = message.Vi
				v.Hw = message.Hw
				if message.Dt != "" {
					v.Dt = message.Dt
				}
			}

			c.isRegistered = true
			c.registeredAt = time.Now().UnixNano()
			// 是否发送上线消息
			isSendOnlineMsg = true
			// 用户注册到Hub
			c.hub.register <- c
			// 退出registration countdown goroutine
			close(c.stopreg)
		}

		// 推送上线消息
		if isSendOnlineMsg {
			if c.isUser() {
				userInfo := c.info.(*UserInfo)
				instruction := &UsrOnlineMsg{
					Act:    "8",
					Uid:    c.id,
					Nm:     userInfo.Nickname,
					Sex:    strconv.Itoa(c.info.(*UserInfo).Sex),
					Idt:    strconv.Itoa(c.identity),
					Os:     userInfo.Os,
					Vi:     userInfo.Vi,
					Hw:     userInfo.Hw,
					Sender: c.id,
					Unit:   c.unitId,
				}
				c.hub.inbound <- instruction
			} else if c.isDevice() {
				deviceInfo := c.info.(*DeviceInfo)
				instruction := &DevOnlineMsg{
					Act:    "10",
					Did:    c.id,
					Nm:     c.id,
					Dt:     deviceInfo.Dt,
					Vi:     deviceInfo.Vi,
					Hw:     deviceInfo.Hw,
					Sender: c.id,
					Unit:   c.unitId,
				}
				c.hub.inbound <- instruction
			}
		}
	case *LocalRegMsg:
		// "本地注册消息"只能由"本地中控"发送
		if c.isLocalControl() {
			if message.Act == "2" {
				// 新增本地终端
				for _, item := range message.Usr {
					item.RegisteredAt = time.Now().Unix()
					c.localUsers.Add(*item)
					// 推送上线消息
					instruction := &UsrOnlineMsg{
						Act:    "8",
						Uid:    item.Uid,
						Nm:     item.Nm,
						Sex:    item.Sex,
						Idt:    item.Idt,
						Os:     item.Os,
						Vi:     item.Vi,
						Hw:     item.Hw,
						Sender: c.id,
						Unit:   c.unitId,
					}
					c.hub.inbound <- instruction
				}
				for _, item := range message.Dev {
					item.RegisteredAt = time.Now().Unix()
					c.localDevices.Add(*item)
					// 推送上线消息
					instruction := &DevOnlineMsg{
						Act:    "10",
						Did:    item.Did,
						Nm:     item.Did,
						Dt:     item.Dt,
						Vi:     item.Vi,
						Hw:     item.Hw,
						Sender: c.id,
						Unit:   c.unitId,
					}
					c.hub.inbound <- instruction
				}
			} else if message.Act == "3" {
				// 清空已有本地终端，将消息体里的终端作为新的终端
				c.localUsers = new(LocalUserSet)
				c.localDevices = new(LocalDeviceSet)
				for _, item := range message.Usr {
					item.RegisteredAt = time.Now().Unix()
					c.localUsers.Add(*item)
					// 推送上线消息
					instruction := &UsrOnlineMsg{
						Act:    "8",
						Uid:    item.Uid,
						Nm:     item.Nm,
						Sex:    item.Sex,
						Idt:    item.Idt,
						Os:     item.Os,
						Vi:     item.Vi,
						Hw:     item.Hw,
						Sender: c.id,
						Unit:   c.unitId,
					}
					c.hub.inbound <- instruction
				}
				for _, item := range message.Dev {
					item.RegisteredAt = time.Now().Unix()
					c.localDevices.Add(*item)
					// 推送上线消息
					instruction := &DevOnlineMsg{
						Act:    "10",
						Did:    item.Did,
						Nm:     item.Did,
						Dt:     item.Dt,
						Vi:     item.Vi,
						Hw:     item.Hw,
						Sender: c.id,
						Unit:   c.unitId,
					}
					c.hub.inbound <- instruction
				}
			}
		} else {
			log.Printf("[%s] Not local control, discard message.\n", c.id)
			return
		}
	case *OrdinaryMsg:
		if message.To == "" {
			log.Printf("[%s] No field 'to', discard message.\n")
			c.notice("No field 'to', discard message.")
			return
		}
		message.Sender = c.id
		message.Unit = c.unitId
		c.hub.inbound <- message
	case *ModStatusMsg:
		if message.To == "" {
			log.Printf("[%s] No field 'to', discard message.\n", c.id)
			c.notice("No field 'to', discard message.")
			return
		}

		// 获取当前单元模块
		if message.Mod != "" {
			c.unitInfo.Curmod = message.Mod
		}

		// 记录状态指令历史
		message.CreatedAt = time.Now().Unix()
		storeData, _ := json.Marshal(message)
		modInsHisKey := fmt.Sprintf(modInsHistoryKeyFormat, c.unitId, c.unitInfo.SceneId, c.unitInfo.Curmod)
		if _, err = c.redconn.Do("RPUSH", modInsHisKey, string(storeData)); err != nil {
			c.notice("Failed to rpush " + modInsHisKey)
			c.logout("Failed to rpush " + modInsHisKey)
			return
		}
		log.Printf("[%s] RPUSH %s %s", c.id, modInsHisKey, string(storeData))

		// 更新当前单元模块状态
		modInsKey := fmt.Sprintf(modInsKeyFormat, c.unitId, c.unitInfo.SceneId)
		ret, err := redis.Bytes(c.redconn.Do("HGET", modInsKey, c.unitInfo.Curmod))
		if err != nil && err != redis.ErrNil {
			log.Printf("[%s] Failed to hget %s, error: %s\n", c.id, modInsKey, err)
			c.logout("Failed to hget " + modInsKey)
			return
		}
		// log.Printf("debug........... %#v, %#v\n", string(ret), err)

		if ret == nil {
			if message.Mod == "" || message.To == "" {
				log.Printf("[%s] Field 'mod' or 'to' not exists, can't be init, discard the instruction.\n", c.id)
				c.notice("Field 'mod' or 'to' not exists, can't be init, discard the instruction.")
				return
			}

			message.UpdatedAt = time.Now().Unix()
			storeData, _ = json.Marshal(message)
			if _, err := c.redconn.Do("HSET", modInsKey, c.unitInfo.Curmod, storeData); err != nil {
				log.Printf("[%s] Failed to hset %s, error: %s\n", c.id, modInsKey, err)
				c.logout("Failed to hset " + modInsKey)
				return
			}
			log.Printf("[%s] HSET %s %s %s", c.id, modInsKey, c.unitInfo.Curmod, storeData)
		} else {
			incrmsg, ok := message.Msg.(map[string]interface{})
			if !ok {
				log.Printf("[%s] field 'msg' not exists, discard the instruction\n", c.id)
				c.notice("field 'msg' not exists, discard the instruction")
				return
			}

			var curstat ModStatusMsg
			if err := json.Unmarshal(ret, &curstat); err != nil {
				log.Printf("[%s] Failed to json.Marshal: %s\n", c.id, err)
				c.notice("Json format error")
				return
			}

			curstat.UpdatedAt = time.Now().Unix()
			curstat.To = message.To

			currmsg, ok := curstat.Msg.(map[string]interface{})
			if !ok || (incrmsg["nm"] != nil && incrmsg["nm"] != currmsg["nm"]) {
				curstat.Msg = message.Msg
			} else {
				for k, v := range incrmsg {
					currmsg[k] = v
				}
				curstat.Msg = currmsg
			}

			storeData, _ = json.Marshal(curstat)
			if _, err := c.redconn.Do("HSET", modInsKey, c.unitInfo.Curmod, storeData); err != nil {
				log.Printf("[%s] Failed to hset %s, error: %s\n", c.id, modInsKey, err)
				c.logout("Failed to hset " + modInsKey)
				return
			}
			log.Printf("[%s] HSET %s %s %s\n", c.id, modInsKey, c.unitInfo.Curmod, storeData)
		}

		message.Sender = c.id
		message.Unit = c.unitId
		c.hub.inbound <- message
	case *UsrOnlineMsg:
	case *UsrOfflineMsg:
		if message.Uid == c.id {
			c.logout("Terminate client")
		} else {
			if c.isLocalControl() {
				c.localUsers.Remove(message.Uid)
			}
		}
		// 广播下线通知
		message.Sender = c.id
		message.Unit = c.unitId
		c.hub.inbound <- message
	case *DevOnlineMsg:
	case *DevOfflineMsg:
		if message.Did == c.id {
			c.logout("Terminate client")
		} else {
			if c.isLocalControl() {
				c.localDevices.Remove(message.Did)
			}
		}

		// 广播下线通知
		message.Sender = c.id
		message.Unit = c.unitId
		c.hub.inbound <- message
	case *UnitControlMsg:
		stat := message.Msg.(map[string]interface{})["stat"]
		if stat == "1" {
			// 开始课程
			sceneInfo := map[string]interface{}{
				"unit_id":    c.unitId,
				"scene_id":   c.unitInfo.SceneId,
				"start_time": time.Now().Unix(),
			}
			b, err := json.Marshal(sceneInfo)
			if err != nil {
				c.logout(err.Error())
				return
			}
			sceneKey := fmt.Sprintf(sceneKeyFormat, c.unitId, c.unitInfo.SceneId)
			if _, err := c.redconn.Do("SET", sceneKey, string(b)); err != nil {
				c.logout(err.Error())
				return
			}
			c.log(fmt.Sprintf("SET %s %s", sceneKey, string(b)))
			log.Printf("[%s] SET %s %s\n", c.id, sceneKey, string(b))
		} else if stat == "2" {
			// TODO 结束单元逻辑
			// 记录单元场景结束时间
			sceneKey := fmt.Sprintf(sceneKeyFormat, c.unitId, c.unitInfo.SceneId)
			res, err := redis.Bytes(c.redconn.Do("GET", sceneKey))
			if err != nil && err != redis.ErrNil {
				c.logout(err.Error())
				return
			}
			sceneInfo := make(map[string]interface{})
			if res == nil {
				sceneInfo = map[string]interface{}{
					"unit_id":  c.unitId,
					"scene_id": c.unitInfo.SceneId,
					"end_time": time.Now().Unix(),
				}
			} else {
				if err := json.Unmarshal(res, &sceneInfo); err != nil {
					c.logout(err.Error())
					return
				}
				sceneInfo["end_time"] = time.Now().Unix()
			}
			b, err := json.Marshal(sceneInfo)
			if err != nil {
				c.logout(err.Error())
				return
			}
			if _, err := c.redconn.Do("SET", sceneKey, string(b)); err != nil {
				c.logout(err.Error())
				return
			}
			c.log(fmt.Sprintf("SET %s %s", sceneKey, string(b)))
			log.Printf("[%s] SET %s %s\n", c.id, sceneKey, string(b))

			// 自增场景id
			sceneIdKey := fmt.Sprintf(sceneIdKeyFormat, c.unitId)
			if _, err := c.redconn.Do("INCR", sceneIdKey); err != nil {
				c.logout(err.Error())
				return
			}
			log.Printf("[%s] INCR %s\n", c.id, sceneIdKey)

			// 结束课程
			c.logout("Terminate, end course")
			return
		}
		// 广播课程状态消息
		message.Sender = c.id
		message.Unit = c.unitId
		c.hub.inbound <- message
	case *PullInkMsg:
		message.Sender = c.id
		message.Unit = c.unitId
		c.hub.inbound <- message
	case *EndPullInkMsg:
		message.Sender = c.id
		message.Unit = c.unitId
		c.hub.inbound <- message
	case *ChatTextMsg:
		text := message.Msg.(map[string]interface{})["c"].(string)
		if len(text) > 100 {
			c.notice("chat message too long")
			return
		}

		// 广播文字聊天消息
		message.CreatedAt = time.Now().Unix()
		message.Sender = c.id
		message.Unit = c.unitId
		c.hub.inbound <- message

		// 持久化文字聊天消息
		storedMsg, err := json.Marshal(message)
		if err != nil {
			log.Printf("[%s] Failed to json.Marshal: %s\n", c.id, err)
			c.notice("Json format error")
			return
		}
		chatKey := fmt.Sprintf(chatKeyFormat, c.unitId, c.unitInfo.SceneId)
		if _, err := c.redconn.Do("RPUSH", chatKey, string(storedMsg)); err != nil {
			log.Printf("[%s] Failed to RPUSH chat message\n", c.id)
			c.notice("Failed to RPUSH chat message")
			return
		}
		log.Printf("[%s] RPUSH %s %s\n", c.id, chatKey, string(storedMsg))
	default:
	}

	return
}
