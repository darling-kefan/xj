package ndscloud

import (
	"log"
	"strconv"
	"time"
)

// 负责从客户端接收消息，并解析、处理、转发等
// 主要包括：json消息(文本消息)处理器 和 笔迹流消息处理器
func (c *Client) process(raw []byte) {
	unmarshalRaw, err := UnmarshalMessage(raw)
	if err != nil {
		log.Println(err)
		return
	}

	switch message := unmarshalRaw.(type) {
	case *RegMsg:
		// 判断注册消息是否和客户端身份匹配
		if (c.isDevice() && message.Dt == "") || (c.isUser() && message.Dt != "") {
			log.Println("bad registration message format: user connect!")
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
		}

		// 推送上线消息
		if isSendOnlineMsg {
			if c.isUser() {
				userInfo := c.info.(*UserInfo)
				log.Printf("%#v\n", userInfo)
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
				log.Printf("%#v\n", deviceInfo)
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
		}
	case *OrdinaryMsg:
		if message.To == "" {
			log.Printf("[%s] No field 'to', discard message.\n", c.id)
		}
		message.Sender = c.id
		message.Unit = c.unitId
		c.hub.inbound <- message
	case *ModStatusMsg:

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
		} else if stat == "2" {
			// 结束课程
			c.logout("Terminate, end course")
		}
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
		message.Sender = c.id
		message.Unit = c.unitId
		c.hub.inbound <- message
	default:
	}

	return
}
