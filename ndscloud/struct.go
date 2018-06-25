package ndscloud

import (
	"encoding/json"
	"time"
)

// Token关联信息
type TokenInfo struct {
	DeviceInfo
	UserInfo
}

// --------------------------------------------------------------------

// 用户详情
type UserInfo struct {
	Sub        string          `json:"sub"`
	Uid        string          `json:"uid"`
	Tpuid_1    string          `json:"tpuid_1,omitempty"`
	Tpuid_2    string          `json:"tpuid_2,omitempty"`
	Tpuid_3    string          `json:"tpuid_3,omitempty"`
	Scope      string          `json:"scope"`
	Username   string          `json:"username"`
	Phone      string          `json:"phone"`
	Email      string          `json:"email"`
	Nickname   string          `json:"nickname"`
	Name       string          `json:"name"`
	Sex        int             `json:"sex"`
	Birthdate  string          `json:"birthdate"`
	Avatar     string          `json:"avatar,omitempty"`
	Tag        string          `json:"tag,omitempty"`
	Signature  string          `json:"signature,omitempty"`
	Permission Permission      `json:"permission,omitempty"`
	District   District        `json:"district,omitempty"`
	Faces      Faces           `json:"faces,omitempty"`
	Attribute  json.RawMessage `json:"attribute,omitempty"`
	Os         string          `json:"-"`
	Vi         string          `json:"-"`
	Hw         string          `json:"-"`
}

// 机构权限
type Permission struct {
	Organization []Organization `json:"organization,omitempty"`
}

// 所属机构及角色
type Organization struct {
	Oid  int `json:"oid"`
	Role int `json:"role"`
	//Attribute Attribute `json:"attribute,omitempty"`
}

// 用户机构属性
type Attribute struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	//Sub  Attribute `json:"sub"`
}

// 地区
type District struct {
	Country  Region `json:"country"`
	Province Region `json:"province"`
	City     Region `json:"city"`
	County   Region `json:"county"`
}

// 行政区域公有属性
type Region struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

// 人脸识别
type Faces struct {
	Status    int       `json:"status"`
	Upstatus  int       `json:"upstatus"`
	UpdatedAt time.Time `json:"updated_at"`
}

// --------------------------------------------------------------------

// 设备详情
type DeviceInfo struct {
	ClientId   string     `json:"client_id"`
	Scope      string     `json:"scope"`
	Permission Permission `json:"permission,omitempty"`
	Config     Config     `json:"config,omitempty"`
	Dt         string     `json:"-"`
	Vi         string     `json:"-"`
	Hw         string     `json:"-"`
}

type Config struct {
	Device Device `json:"device"`
}

type Device struct {
	DeviceType     int    `json:"device_type"`
	ClassroomId    string `json:"classroom_id"`
	ClassroomTitle string `json:"classroom_title"`
}

// --------------------------------------------------------------------

// 房间信息
type ClassroomInfo struct {
	Id    string `json:"id"`
	Title string `json:"title"`
}

// 单元信息
type UnitInfo struct {
	CourseId       string          `json:"course_id"`
	UnitId         string          `json:"unit_id"`
	Type           string          `json:"type"`
	Title          string          `json:"title"`
	Desc           string          `json:"desc"`
	Cover          string          `json:"cover"`
	Status         string          `json:"status"`
	EventId        string          `json:"event_id"`
	StartTime      string          `json:"start_time"`
	EndTime        string          `json:"end_time"`
	CreatedAt      string          `json:"create_at"`
	Classroom      []ClassroomInfo `json:"classroom,omitempty"`
	ClassStartTime time.Time       `json:"class_start_time,omitempty"`
	ClassEndTime   time.Time       `json:"class_end_time,omitempty"`
	SceneId        int             `json:"-"`
	Curmod         string          `json:"-"`
}

// --------------------------------------------------------------------

// 用户在课程中的身份
type CourseIdentity struct {
	Uid      string
	Identity string
	CourseId string
}
