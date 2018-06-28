package ndscloud

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	//"github.com/darling-kefan/xj/config"
	"github.com/darling-kefan/xj/helper"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/websocket"
)

// 输出json
func outputJson(c *gin.Context, errcode int, errmsg string, data interface{}) {
	jsonData := gin.H{
		"errcode": errcode,
		"errmsg":  errmsg,
	}
	if data != nil {
		jsonData = gin.H{
			"errcode": errcode,
			"errmsg":  errmsg,
			"data":    data,
		}
	}
	c.JSON(http.StatusOK, jsonData)
}

// TODO 如何支持JSONP???
func ServeUsers(hub *Hub, c *gin.Context) {
	// 判断课程是否公开/免费
	unitId := c.Param("unit_id")
	ok, err := isPublicAndPremium(unitId)
	if err != nil {
		outputJson(c, 1, err.Error(), nil)
		return
	}
	// 验证token
	if !ok {
		token := c.Query("token")
		if token == "" {
			outputJson(c, 1, "missing param token", nil)
			return
		}
		if _, err := getTokenInfo(token); err != nil {
			outputJson(c, 1, err.Error(), nil)
			return
		}
	}

	type User struct {
		Type          int         `json:"type"`
		Id            string      `json:"id"`
		Name          string      `json:"name"`
		VideoInteract int         `json:"videointeract"`
		HandWrite     int         `json:"handwrite"`
		OnlineAt      int64       `json:"online_at"`
		Classroom     string      `json:"classroom"`
		UserInfo      interface{} `json:"userinfo"`
	}
	type Device struct {
		Type          int    `json:"type"`
		Id            string `json:"id"`
		Name          string `json:"name"`
		VideoInteract int    `json:"videointeract"`
		HandWrite     int    `json:"handwrite"`
		OnlineAt      int64  `json:"online_at"`
		Classroom     string `json:"classroom"`
	}

	clients := make([]interface{}, 0)
	for _, client := range hub.list() {
		switch {
		case client.isLocalControl():
			deviceDetail := client.info.(*DeviceInfo)
			dt, _ := strconv.Atoi(deviceDetail.Dt)
			vi, _ := strconv.Atoi(deviceDetail.Vi)
			hw, _ := strconv.Atoi(deviceDetail.Hw)
			device := &Device{
				Type:          dt,
				Id:            client.id,
				Name:          deviceDetail.ClientId,
				VideoInteract: vi,
				HandWrite:     hw,
				OnlineAt:      client.registeredAt,
				Classroom:     client.unitInfo.Classroom[0].Id,
			}
			clients = append(clients, device)

			localUsers := client.localUsers
			for _, userItem := range localUsers.List() {
				sex, _ := strconv.Atoi(userItem.Sex)
				idt, _ := strconv.Atoi(userItem.Idt)
				vi, _ := strconv.Atoi(userItem.Vi)
				hw, _ := strconv.Atoi(userItem.Hw)
				userInfo := make(map[string]interface{})
				userInfo["sex"] = sex
				userInfo["avatar"] = ""
				userInfo["identity"] = idt
				localUser := &User{
					Type:          0,
					Id:            userItem.Uid,
					Name:          userItem.Nm,
					VideoInteract: vi,
					HandWrite:     hw,
					OnlineAt:      userItem.RegisteredAt,
					Classroom:     client.unitInfo.Classroom[0].Id,
					UserInfo:      userInfo,
				}
				clients = append(clients, localUser)
			}

			localDevices := client.localDevices
			for _, deviceItem := range localDevices.List() {
				dt, _ := strconv.Atoi(deviceItem.Dt)
				vi, _ := strconv.Atoi(deviceItem.Vi)
				hw, _ := strconv.Atoi(deviceItem.Hw)
				localDevice := &Device{
					Type:          dt,
					Id:            deviceItem.Did,
					Name:          deviceItem.Nm,
					VideoInteract: vi,
					HandWrite:     hw,
					OnlineAt:      deviceItem.RegisteredAt,
					Classroom:     client.unitInfo.Classroom[0].Id,
				}
				clients = append(clients, localDevice)
			}
		case client.isUser():
			userDetail := client.info.(*UserInfo)
			userAttr := make(map[string]interface{})
			userAttr["sex"] = userDetail.Sex
			userAttr["avatar"] = userDetail.Avatar
			userAttr["identity"] = client.identity
			vi, _ := strconv.Atoi(userDetail.Vi)
			hw, _ := strconv.Atoi(userDetail.Hw)
			user := &User{
				Type:          0,
				Id:            client.id,
				Name:          userDetail.Name,
				VideoInteract: vi,
				HandWrite:     hw,
				OnlineAt:      client.registeredAt,
				Classroom:     client.unitInfo.Classroom[0].Id,
				UserInfo:      userAttr,
			}
			clients = append(clients, user)
		case client.isDevice():
			deviceDetail := client.info.(*DeviceInfo)
			dt, _ := strconv.Atoi(deviceDetail.Dt)
			vi, _ := strconv.Atoi(deviceDetail.Vi)
			hw, _ := strconv.Atoi(deviceDetail.Hw)
			device := &Device{
				Type:          dt,
				Id:            client.id,
				Name:          deviceDetail.ClientId,
				VideoInteract: vi,
				HandWrite:     hw,
				OnlineAt:      client.registeredAt,
				Classroom:     client.unitInfo.Classroom[0].Id,
			}
			clients = append(clients, device)
		}
	}

	data := make(map[string]interface{})
	data["total"] = len(clients)
	data["list"] = clients
	outputJson(c, 0, "OK", data)
}

// 获取所有单元场景
func ServeScenes(c *gin.Context) {
	c.String(http.StatusOK, "Hello scenes")
}

// 获取单元最新模块状态
func ServeModStatus(c *gin.Context) {
	// 判断课程是否公开/免费
	unitId := c.Param("unit_id")
	moduleId := c.Param("module_id")
	ok, err := isPublicAndPremium(unitId)
	if err != nil {
		outputJson(c, 1, err.Error(), nil)
		return
	}
	// 验证token
	if !ok {
		token := c.Query("token")
		if token == "" {
			outputJson(c, 1, "missing param token", nil)
			return
		}
		if _, err := getTokenInfo(token); err != nil {
			outputJson(c, 1, err.Error(), nil)
			return
		}
	}

	// 创建redis连接
	redconn, err := connectRedis()
	if err != nil {
		outputJson(c, 1, err.Error(), nil)
		return
	}

	// 获取最新场景id
	sceneIdKey := fmt.Sprintf(sceneIdKeyFormat, unitId)
	sceneId, err := redis.Int(redconn.Do("GET", sceneIdKey))
	if err == redis.ErrNil {
		outputJson(c, 0, "OK", gin.H{
			"total": 0,
			"list":  make([]interface{}, 0),
		})
		return
	}
	if err != nil {
		outputJson(c, 1, err.Error(), nil)
		return
	}

	modInsKey := fmt.Sprintf(modInsKeyFormat, unitId, sceneId)
	modinses, err := redis.StringMap(redconn.Do("HGETALL", modInsKey))
	if err != nil {
		outputJson(c, 1, err.Error(), nil)
		return
	}

	modMap := make([]interface{}, 0)
	for mod, v := range modinses {
		var modstat ModStatusMsg
		if err := json.Unmarshal([]byte(v), &modstat); err != nil {
			outputJson(c, 1, err.Error(), nil)
			return
		}
		item := map[string]interface{}{
			"id":         "1", // TODO 写死
			"mod":        mod,
			"msg":        modstat.Msg,
			"updated_at": time.Unix(modstat.UpdatedAt, 0).Format("2006-01-02 15:04:05"),
		}

		if moduleId == "" || moduleId == modstat.Mod {
			modMap = append(modMap, item)
		}
	}

	// 按更新时间降序排列
	sort.Slice(modMap, func(i, j int) bool {
		ti, _ := time.Parse("2006-01-02 15:04:05", modMap[i].(map[string]interface{})["updated_at"].(string))
		tj, _ := time.Parse("2006-01-02 15:04:05", modMap[j].(map[string]interface{})["updated_at"].(string))
		return ti.After(tj)
	})

	outputJson(c, 0, "OK", modMap)
}

// 获取模块状态指令历史
func ServeModList(c *gin.Context) {
	// 判断课程是否公开/免费
	unitId := c.Param("unit_id")
	ok, err := isPublicAndPremium(unitId)
	if err != nil {
		outputJson(c, 1, err.Error(), nil)
		return
	}
	// 验证token
	if !ok {
		token := c.Query("token")
		if token == "" {
			outputJson(c, 1, "missing param token", nil)
			return
		}
		if _, err := getTokenInfo(token); err != nil {
			outputJson(c, 1, err.Error(), nil)
			return
		}
	}

	// 创建redis连接
	redconn, err := connectRedis()
	if err != nil {
		outputJson(c, 1, err.Error(), nil)
		return
	}

	sceneId := 0
	if c.Param("scene_id") != "" {
		sceneId, err = strconv.Atoi(c.Param("scene_id"))
		if err != nil {
			outputJson(c, 1, err.Error(), nil)
			return
		}
	}
	// 获取最新场景id
	if sceneId == 0 {
		sceneIdKey := fmt.Sprintf(sceneIdKeyFormat, unitId)
		sceneId, err = redis.Int(redconn.Do("GET", sceneIdKey))
		if err == redis.ErrNil {
			outputJson(c, 0, "OK", gin.H{
				"total": 0,
				"list":  make([]interface{}, 0),
			})
			return
		}
		if err != nil {
			outputJson(c, 1, err.Error(), nil)
			return
		}
	}

	// 检索该单元场景下的所有模块
	iter := 0
	match := fmt.Sprintf("nc:ins:mod:his:%s:%d:*", unitId, sceneId)
	modkeys := make([]string, 0)
	for {
		res, err := redis.Values(redconn.Do("SCAN", iter, "MATCH", match, "COUNT", 1))
		if err != nil {
			outputJson(c, 1, err.Error(), nil)
			return
		}
		for _, v := range res[1].([]interface{}) {
			modkeys = append(modkeys, string(v.([]byte)))
		}
		iter, _ = strconv.Atoi(string(res[0].([]byte)))
		if iter == 0 {
			break
		}
	}
	log.Printf("%#v\n", modkeys)

	statmods := make(map[string][]map[string]interface{})
	modtimes := make([]map[string]interface{}, 0)
	for _, modkey := range modkeys {
		res, err := redis.ByteSlices(redconn.Do("lrange", modkey, 0, -1))
		if err != nil {
			outputJson(c, 1, err.Error(), nil)
			return
		}

		modinses := make([]map[string]interface{}, 0)
		for _, v := range res {
			var modins ModStatusMsg
			_ = json.Unmarshal(v, &modins)
			modItem := map[string]interface{}{
				"id":         "1",
				"mod":        modins.Mod,
				"msg":        modins.Msg,
				"created_at": modins.CreatedAt,
			}
			modinses = append(modinses, modItem)
		}
		statmods[modkey] = modinses
		// 用于排序模块
		modtimes = append(modtimes, map[string]interface{}{
			"key":        modkey,
			"created_at": modinses[0]["created_at"].(int64),
		})
	}

	// 按时间升序排列模块
	sort.Slice(modtimes, func(i, j int) bool {
		return modtimes[i]["created_at"].(int64) < modtimes[j]["created_at"].(int64)
	})

	i := 1
	instructions := make([]map[string]interface{}, 0)
	for _, v := range modtimes {
		modkey := v["key"].(string)
		for _, item := range statmods[modkey] {
			item["id"] = strconv.Itoa(i)
			item["created_at"] = time.Unix(item["created_at"].(int64), 0).Format("2006-01-02 15:04:05")
			instructions = append(instructions, item)
			i = i + 1
		}
	}

	outputJson(c, 0, "OK", gin.H{
		"total": len(instructions),
		"list":  instructions,
	})
}

// 文字消息列表
func ServeChats(c *gin.Context) {
	// 判断课程是否公开/免费
	unitId := c.Param("unit_id")
	ok, err := isPublicAndPremium(unitId)
	if err != nil {
		outputJson(c, 1, err.Error(), nil)
		return
	}
	// 验证token
	if !ok {
		token := c.Query("token")
		if token == "" {
			outputJson(c, 1, "missing param token", nil)
			return
		}
		if _, err := getTokenInfo(token); err != nil {
			outputJson(c, 1, err.Error(), nil)
			return
		}
	}

	chatId := 0
	if c.Query("chat_id") != "" {
		chatId, err = strconv.Atoi(c.Query("chat_id"))
		if err != nil {
			outputJson(c, 1, err.Error(), nil)
			return
		}
	}
	limit := 20
	if c.Query("limit") != "" {
		limit, err = strconv.Atoi(c.Query("limit"))
		if err != nil {
			outputJson(c, 1, err.Error(), nil)
			return
		}
	}

	// 创建redis连接
	redconn, err := connectRedis()
	if err != nil {
		outputJson(c, 1, err.Error(), nil)
		return
	}

	sceneId := 0
	if c.Param("scene_id") != "" {
		sceneId, err = strconv.Atoi(c.Param("scene_id"))
		if err != nil {
			outputJson(c, 1, err.Error(), nil)
			return
		}
	}
	if sceneId == 0 {
		// 获取最新场景id
		sceneIdKey := fmt.Sprintf(sceneIdKeyFormat, unitId)
		sceneId, err = redis.Int(redconn.Do("GET", sceneIdKey))
		if err == redis.ErrNil {
			outputJson(c, 0, "OK", gin.H{
				"total": 0,
				"list":  make([]interface{}, 0),
			})
			return
		}
		if err != nil {
			outputJson(c, 1, err.Error(), nil)
			return
		}
		// 如果课程在进行中则默认读取当前场景下的聊天记录，否则默认取上一个场景下的聊天记录
		token, err := helper.AccessToken(redconn, "client_credentials", nil)
		if err != nil {
			outputJson(c, 1, err.Error(), nil)
			return
		}
		unitInfo, err := getUnitInfo(token, unitId)
		if err != nil {
			outputJson(c, 1, err.Error(), nil)
			return
		}
		if unitInfo.Status != "1" {
			sceneId = sceneId - 1
		}
	}

	chatKey := fmt.Sprintf(chatKeyFormat, unitId, sceneId)
	// 获取列表长度
	count, err := redis.Int(redconn.Do("LLEN", chatKey))
	if err != nil {
		outputJson(c, 1, err.Error(), nil)
	}
	if count == 0 {
		outputJson(c, 0, "OK", gin.H{
			"total": 0,
			"list":  make([]interface{}, 0),
		})
		return
	}

	// 设置聊天记录的开始位置/结束位置
	var sp, ep int
	if chatId == 0 {
		ep = count - 1
	} else {
		ep = chatId - 1
	}
	sp = ep - limit + 1
	if sp < 0 {
		sp = 0
	}
	chatmsgs := make([]map[string]interface{}, 0)
	if ep >= sp && ep >= 0 {
		res, err := redis.ByteSlices(redconn.Do("lrange", chatKey, sp, ep))
		if err != nil {
			outputJson(c, 1, err.Error(), nil)
			return
		}
		length := len(res)
		if length > 0 {
			for k, v := range res {
				msgId := ep - length + k + 2
				var chatText ChatTextMsg
				if err := json.Unmarshal(v, &chatText); err != nil {
					outputJson(c, 1, err.Error(), nil)
					return
				}
				nva := map[string]interface{}{
					"chat_id":    msgId,
					"from":       chatText.From,
					"msg":        chatText.Msg,
					"created_at": time.Unix(chatText.CreatedAt, 0).Format("2006-01-02 15:04:05"),
				}
				chatmsgs = append(chatmsgs, nva)
			}
		}
	}

	// 排序
	if c.Query("sort") == "desc" {
		for i := len(chatmsgs)/2 - 1; i >= 0; i-- {
			opp := len(chatmsgs) - 1 - i
			chatmsgs[i], chatmsgs[opp] = chatmsgs[opp], chatmsgs[i]
		}
	}

	outputJson(c, 0, "OK", gin.H{
		"total": len(chatmsgs),
		"list":  chatmsgs,
	})
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

//func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
func ServeWs(hub *Hub, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		log.Println(err)
		return
	}

	token := c.Query("token")
	if token == "" {
		conn.WriteMessage(websocket.TextMessage, []byte("token is empty."))
		conn.Close()
		return
	}

	unitId := c.Param("unit_id")
	if unitId == "" {
		conn.WriteMessage(websocket.TextMessage, []byte("unit_id is empty."))
		conn.Close()
		return
	}

	redconn, err := connectRedis()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		conn.Close()
		return
	}

	// create new client, then add it to the hub.
	client, err := NewClient(token, unitId, redconn, conn, hub)
	if err != nil {
		log.Println(err)
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		conn.Close()
		redconn.Close()
		return
	}

	// Forced login
	client.forceLogin()

	// Registration countdown.
	// Close the client connection if registration is not submitted within two seconds.
	go func() {
		defer func() {
			log.Println("End register countdown")
		}()

		tc := time.After(10 * time.Second)
		select {
		case <-tc:
			if client == hub.get(client.id) && !client.isRegistered {
				hub.unregister <- client
			}
		case <-client.stopreg:
			// 如果客户端已经下线，则退出倒计时goroutine
			return
		}
	}()

	go client.readPump()
	go client.writePump()
}
