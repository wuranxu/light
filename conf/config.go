package conf

import (
	"encoding/json"
	"errors"
	sp "github.com/bitly/go-simplejson"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"time"
)

var Conf = new(Config)

var (
	ConfigError = errors.New("config is not found")
	DEFAULTENV  = "DEV"
)

type EtcdConfig struct {
	Endpoints   []string      `json:"endpoints"`
	DialTimeout time.Duration `json:"dial-timeout"`
	Scheme      string        `json:"scheme"`
}

type SqlConfig struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	Timeout  uint   `json:"timeout"`
	LogMode  bool   `json:"log_mode"`
}

type Config struct {
	Etcd     EtcdConfig `json:"etcd"`
	Database SqlConfig  `json:"database"`
	Scheme   string     `json:"scheme"`
}

type YamlConfig struct {
	Service string        `yaml:"service"`
	Version string        `yaml:"version"`
	Port    int           `yaml:"port"`
	Method  map[string]Md `yaml:"method"`
}

type Md struct {
	NoAuth bool `yaml:"no_auth"`
}

func ParseConfig(filepath string, object interface{}, env string) error {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}
	if env == "" {
		// 未配置环境变量, 默认读取dev配置
		env = DEFAULTENV
	}
	result, err := sp.NewJson(data)
	if err != nil {
		return err
	}
	if result.Get(env).Interface() == nil {
		return ConfigError
	}
	if data, err = result.Get(env).Encode(); err != nil {
		return err
	}
	return json.Unmarshal(data, object)
}

func ParseYaml(filepath string, out interface{}) error {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, out)
}
