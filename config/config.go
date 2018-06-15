package config

import (
	"io/ioutil"
	"log"

	"github.com/BurntSushi/toml"
)

type TomlConfig struct {
	Version string
	Common  CommonInfo
	Redis   RedisInfo
	OAuth2  OAuth2
	Api     Api
	Cc      Cc
	Stat    Stat
	SSDB    SSDB
	MySQL   map[string]MySQL
	Kafka   Kafka
}

type CommonInfo struct {
	Pmspath        string
	NdscloudScheme string
	NdscloudDomain string
	StatHost       string
	CanvasHost     string
}

type RedisInfo struct {
	Host string
	Port int
	Auth string
	DB   int
}

type OAuth2 struct {
	ClientId       string
	ClientSecret   string
	AuthorizeApi   string
	TokenApi       string
	TokeninfoApi   string
	ClientTokenKey string
	PasswdTokenKey string
}

type Api struct {
	Domain       string
	UnitInfo     string
	UnitIdentity string
}

type Cc struct {
	Domain string
	Wsapi  string
}

type Stat struct {
	Districtdb string
	Cityipdb   string
}

type SSDB struct {
	Host string
	Port int
	Auth string
}

type Kafka struct {
	Servers string
}

type MySQL struct {
	Host     string
	Port     int
	Username string
	Password string
	Dbname   string
}

var Config *TomlConfig

func Load(filepath string) {
	if filepath == "" {
		filepath = "./.ndscloud.toml"
	} else {
		filepath = filepath + "/.ndscloud.toml"
	}
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatal(err)
	}

	var conf TomlConfig
	if _, err := toml.Decode(string(b), &conf); err != nil {
		log.Fatal(err)
	}
	Config = &conf
}
