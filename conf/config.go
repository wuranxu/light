package conf

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

var Conf = new(Config)

type EtcdConfig struct {
	Endpoints   []string `yaml:"endpoints"`
	DialTimeout int64    `yaml:"dial_timeout"`
	Scheme      string   `yaml:"scheme"`
	Username    string   `yaml:"username"`
	Password    string   `yaml:"password"`
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
	Etcd EtcdConfig `yaml:"etcd"`
	//Database SqlConfig  `json:"database"`
	Scheme string `yaml:"scheme"`
}

type YamlConfig struct {
	Service string        `yaml:"service"`
	Version string        `yaml:"version"`
	Port    int           `yaml:"port"`
	Method  map[string]Md `yaml:"method"`
}

type Md struct {
	Authorization bool `yaml:"authorization"`
}

func ParseConfig(filepath string, cfg interface{}) error {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, cfg)
}

func Init(filepath string) error {
	return ParseConfig(filepath, Conf)
}
