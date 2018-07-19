package protocol

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// -----------------------------------------------------------------
// Json时间转换
// -----------------------------------------------------------------
type CustomTime struct {
	time.Time
}

const CustomTimeFormat = "2006-01-02 15:04:05"

func (ct *CustomTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		ct.Time = time.Time{}
		return
	}
	ct.Time, err = time.Parse(CustomTimeFormat, s)
	return
}

func (ct *CustomTime) MarshalJSON() ([]byte, error) {
	if ct.Time == (time.Time{}) {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", ct.Format(CustomTimeFormat))), nil
}

// 日志消息头
type LogHeader struct {
	Topic     string `json:"-"` // Kafka topic
	Partition string `json:"-"` // Kafka partition
	Offset    string `json:"-"` // Kafka partition offset
}

// -----------------------------------------------------------------
// 源日志消息格式
// -----------------------------------------------------------------

// Mtype用于标识日志类型
type Mtype string

const (
	LOG_ORG_USER_BIND    Mtype = "1"  // 用户和机构绑定/解绑日志
	LOG_USER_LOGIN       Mtype = "11" // 用户登录日志
	LOG_COURSE           Mtype = "21" // 课程日志
	LOG_COURSE_USER_BIND Mtype = "22" // 课程和用户绑定/解绑日志
	LOG_COURSEWARE       Mtype = "41" // 课件日志
	LOG_ORDER            Mtype = "51" // 订单日志
)

// 日志消息
type LogMsg struct {
	LogHeader
	Mtype Mtype `json:"mtype"`

	Oid    string `json:"oid,omitempty"`
	Act    string `json:"act,omitempty"`
	Sid    string `json:"sid,omitempty"`
	Subkey string `json:"subkey,omitempty"`
	Value  string `json:"value,omitempty"`

	Uid      string `json:"uid,omitempty"`
	Nickname string `json:"nickname,omitempty"`
	IP       string `json:"ip,omitempty"`

	Filetype string `json:"filetype,omitempty"`
	Filesize string `json:"filesize,omitempty"`

	CreatedAt *CustomTime `json:"created_at"`
}

func DecodeLogMsg(r io.Reader) (logMsg *LogMsg, err error) {
	logMsg = new(LogMsg)
	if err = json.NewDecoder(r).Decode(logMsg); err != nil {
		return
	}
	return
}

// -----------------------------------------------------------------
// 日志统计因子格式
// -----------------------------------------------------------------

// Stype用于标识统计因子
type Stype string

const (
	STAT_COUNT_USER            Stype = "1"  // 用户总数
	STAT_COUNT_NEW_USER        Stype = "2"  // 新增用户数
	STAT_COUNT_LOGIN           Stype = "3"  // 用户登录数
	STAT_COUNT_LOGIN_DISTRICT  Stype = "4"  // 每天登录位置数
	STAT_COUNT_LOGIN_TEACHER   Stype = "5"  // 老师登录数
	STAT_COUNT_LOGIN_STUDENT   Stype = "6"  // 学生登录数
	STAT_COUNT_COURSE          Stype = "21" // 课程总数
	STAT_COUNT_COURSE_USER     Stype = "22" // 课程用户数
	STAT_RATING_COURSE         Stype = "23" // 课程评分
	STAT_COUNT_COURSEWARE      Stype = "41" // 课件总数
	STAT_SIZE_COURSEWARE       Stype = "42" // 课件大小(占用空间)
	STAT_COUNT_COURSEWARE_TYPE Stype = "43" // 课件类型数量
	STAT_SIZE_COURSEWARE_TYPE  Stype = "44" // 课件类型大小(占用空间)
	STAT_COUNT_ORDER           Stype = "51" // 订单总数
	STAT_INCOME_ORDER          Stype = "52" // 订单总收入
	STAT_INCOME_NEW_ORDER      Stype = "53" // 每日订单总收入
)

// 日志统计因子，一条日志对应一个StatData
type StatData struct {
	LogHeader
	Factors []*StatFactor
}

type StatFactor struct {
	Stype  Stype   `json:"stype"`
	Oid    string  `json:"oid"`
	Sid    string  `json:"sid"`
	Subkey string  `json:"subkey"`
	Value  float64 `json:"value"`
	Date   string  `json:"date"`
	//Mod    string `json:"mod"`
}
