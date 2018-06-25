package ndscloud

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	//"log"
	"net/http"
	//"strconv"
	"strings"
	//"time"

	"github.com/darling-kefan/xj/config"
	//"github.com/darling-kefan/ndscloud/helper"
)

// 查询单元信息
func getUnitInfo(token string, unitId string) (*UnitInfo, error) {
	var api string = config.Config.Api.Domain + "/v1/units/:unit_id/get?token=:token"
	api = strings.Replace(api, ":unit_id", unitId, -1)
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
		Errcode int      `json:"errcode"`
		Errmsg  string   `json:"errmsg"`
		Data    UnitInfo `json:"data"`
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

// 根据unitid查询用户在课程中的身份
func getUnitidt(token string, unitId string, uid string) (*CourseIdentity, error) {
	var api string = config.Config.Api.Domain + "/v1/units/:unit_id/users/:uid/detail?token=:token"
	api = strings.Replace(api, ":unit_id", unitId, -1)
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

func getTokenInfo(token string) (interface{}, error) {
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

	var flagStruct struct {
		Uid      string `json:"uid"`
		ClientId string `json:"client_id"`
	}
	if err = json.Unmarshal(body, &flagStruct); err != nil {
		return nil, err
	}

	if flagStruct.Uid != "" {
		var userInfo UserInfo
		err = json.Unmarshal(body, &userInfo)
		if err != nil {
			return nil, err
		}
		return &userInfo, nil
	} else if flagStruct.ClientId != "" {
		var deviceInfo DeviceInfo
		err = json.Unmarshal(body, &deviceInfo)
		if err != nil {
			return nil, err
		}
		if deviceInfo.Config.Device.DeviceType != 0 {
			return &deviceInfo, nil
		}
	}

	return nil, errors.New("Invalid token.")
}
