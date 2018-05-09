package helper

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/darling-kefan/xj/config"
	"github.com/gomodule/redigo/redis"
)

// 获取Oauth2 access token
func AccessToken(red redis.Conn, grantType string, params map[string]interface{}) (token string, err error) {
	grantTypes := map[string]bool{
		"password":           true,
		"client_credentials": true,
	}
	if _, ok := grantTypes[grantType]; !ok {
		return "", errors.New("The grant_type is not supported.")
	}

	// 首先取出redis缓存中的token
	var cacheKey string
	switch grantType {
	case "password":
		cacheKey = config.Config.OAuth2.PasswdTokenKey
	case "client_credentials":
		cacheKey = config.Config.OAuth2.ClientTokenKey
	default:
		return "", errors.New("The grant_type is not supported.")
	}
	exists, err := redis.Bool(red.Do("EXISTS", cacheKey))
	if err != nil {
		return "", err
	}
	if exists {
		if token, err = redis.String(red.Do("GET", cacheKey)); err != nil {
			return "", err
		} else {
			return token, nil
		}
	}

	// 如果缓存中不存在，则从接口获取新的token并缓存到redis
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}

	payload := url.Values{}
	switch grantType {
	case "password":
		if _, ok := params["username"]; !ok {
			return "", errors.New("The param username is missing.")
		}
		if _, ok := params["password"]; !ok {
			return "", errors.New("The param password is missing.")
		}
		payload.Set("client_id", config.Config.OAuth2.ClientId)
		payload.Set("client_secret", config.Config.OAuth2.ClientSecret)
		payload.Set("grant_type", "password")
		payload.Set("username", params["username"].(string))
		payload.Set("password", params["password"].(string))
	case "client_credentials":
		payload.Set("client_id", config.Config.OAuth2.ClientId)
		payload.Set("client_secret", config.Config.OAuth2.ClientSecret)
		payload.Set("grant_type", "client_credentials")
	default:
		return "", errors.New("The grant_type is not supported.")
	}

	tokenApi := config.Config.OAuth2.TokenApi
	r, _ := http.NewRequest("POST", tokenApi, strings.NewReader(payload.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(payload.Encode())))

	resp, err := client.Do(r)
	if err != nil {
		return "", nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	//log.Println("request oauth2/token, body:", string(body))
	defer resp.Body.Close()

	tokenInfo := struct {
		Errcode     string `json:"errcode,omitempty"`
		Errmsg      string `json:"errmsg,omitempty"`
		AccessToken string `json:"access_token,omitempty"`
		ExpiresIn   string `json:"expires_in,omitempty"`
		TokenType   string `json:"token_type,omitempty"`
		Scope       string `json:"scope,omitempty"`
	}{}
	err = json.Unmarshal(body, &tokenInfo)
	if err != nil {
		return "", err
	}

	if tokenInfo.Errcode != "" {
		return "", errors.New(tokenInfo.Errmsg)
	}

	_, err = red.Do("SETEX", cacheKey, tokenInfo.ExpiresIn, tokenInfo.AccessToken)
	if err != nil {
		return "", err
	}
	return tokenInfo.AccessToken, nil
}
