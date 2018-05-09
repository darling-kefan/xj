package ndscloud

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/darling-kefan/ndscloud/config"
	//"github.com/darling-kefan/ndscloud/helper"
)

// 用户在课程中的身份
type CourseIdentity struct {
	Uid      int
	Identity int
	CourseId int
}

// 根据unitid查询用户在课程中的身份
func unitidt(token string, unitid string, uid int) (*CourseIdentity, error) {
	var api string = config.Config.Api.Domain + "/v1/units/:unit_id/users/:uid/detail?token=:token"
	api = strings.Replace(api, ":unit_id", unitid, -1)
	api = strings.Replace(api, ":uid", uid, -1)
	api = strings.Replace(api, ":token", token, -1)

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
		Errcode int            `json:"errcode"`
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

// 用户信息

// 设备信息

// Token关联信息
type TokenInfo struct {
}

func tokeninfo(token string) (*TokenInfo, error) {
	var api string = config.Config.OAuth2.TokeninfoApi + "?token=" + token
	resp, err := http.Get(api)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

}
