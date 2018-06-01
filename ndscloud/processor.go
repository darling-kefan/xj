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

		c.vi = message.Vi
		c.hw = message.Hw
		if message.Os != "" {
			c.os = message.Os
		}
		if message.Dt != "" {
			c.dt = message.Dt
		}
		// 注册时间只在第一次注册时设置
		if !c.isRegistered {
			c.isRegistered = true
			c.registeredAt = time.Now().UnixNano()
			isSendOnlineMsg = true
		}

		// 推送上线消息
		if isSendOnlineMsg {
			if c.isUser() {
				instruction := &UsrOnlineMsg{
					Act:    "8",
					Uid:    c.id,
					Nm:     c.info.(*UserInfo).Nickname,
					Sex:    strconv.Itoa(c.info.(*UserInfo).Sex),
					Idt:    strconv.Itoa(c.identity),
					Os:     c.os,
					Vi:     c.vi,
					Hw:     c.hw,
					Sender: c.id,
					Unit:   c.unitId,
				}
				c.hub.inbound <- instruction
			} else if c.isDevice() {
				instruction := &DevOnlineMsg{
					Act:    "10",
					Did:    c.id,
					Nm:     c.id,
					Dt:     c.dt,
					Vi:     c.vi,
					Hw:     c.hw,
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
		c.hub.inbound <- message
	case *ModStatusMsg:

	case *UsrOnlineMsg:

	case *UsrOfflineMsg:

	case *DevOnlineMsg:

	case *DevOfflineMsg:

	case *UnitControlMsg:

	case *PullInkMsg:

	case *EndPullInkMsg:

	case *ChatTextMsg:

	default:
	}

	return
}
