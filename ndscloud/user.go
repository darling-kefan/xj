package ndscloud

import (
	"time"
)

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
}

// 机构权限
type Permission struct {
	Organization []Organization `json:"organization,omitempty"`
}

// 所属机构及角色
type Organization struct {
	Oid  int `json:"oid"`
	Role int `json:"role"`
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
