package config

import (
	"errors"
	"github.com/BurntSushi/toml"
	"github.com/holgerfy/go-pkg/log"
	jsoniter "github.com/json-iterator/go"
	"io/ioutil"
	"strings"
	"sync"
)

type Config struct {
	configs map[string]map[string]interface{}
	once    sync.Once
}

var (
	config           *Config
	json             = jsoniter.Config{EscapeHTML: true, TagKey: "toml"}.Froze()
	ErrNodeNotExists = errors.New("node not exists")
)

func LoadConfig(path []string) {
	config = new(Config).load(path)
}

func GetInstance() *Config {
	return config
}

func (conf *Config) copy(node string, value map[string]interface{}) {
	for key, val := range value {
		if conf.configs[node] == nil {
			conf.configs[node] = make(map[string]interface{})
		}
		conf.configs[node][key] = val
	}
}

func (conf *Config) load(path []string) *Config {
	conf.once.Do(func() {
		conf.configs = make(map[string]map[string]interface{})
		for _, dir := range path {
			rd, err := ioutil.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, fi := range rd {
				if !fi.IsDir() {
					if strings.HasSuffix(dir+fi.Name(), ".toml") {
						var config map[string]interface{}
						if _, err = toml.DecodeFile(dir+fi.Name(), &config); err != nil {
							log.Logger().Fatal(nil, "failed to load config")
						}
						conf.copy(strings.TrimSuffix(fi.Name(), ".toml"), config)
					}
				}
			}
		}
	})
	return conf
}

func (conf *Config) Bind(node, key string, obj interface{}) error {
	nodeVal, ok := conf.configs[node]
	if !ok {
		return nil
	}

	var objVal interface{}
	if key != "" {
		objVal, ok = nodeVal[key]
		if !ok {
			return ErrNodeNotExists
		}
	} else {
		objVal = nodeVal
	}

	return conf.assignment(objVal, obj)
}

func (conf *Config) GetChildConf(node, key, subKey string) (interface{}, error) {
	var confInfo map[string]interface{}
	if err := conf.Bind(node, key, &confInfo); err != nil {
		return nil, err
	}
	val, ok := confInfo[subKey]
	if !ok {
		return nil, ErrNodeNotExists
	}
	return val, nil
}

func (conf *Config) assignment(val, obj interface{}) error {
	data, _ := json.Marshal(val)
	return json.Unmarshal(data, obj)
}
