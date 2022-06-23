package config

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/holgerfy/go-pkg/log"
	jsoniter "github.com/json-iterator/go"
	"io/ioutil"
	"os"
	"path/filepath"
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

func (conf *Config) walk(path string, info os.FileInfo, err error) error {
	if err == nil {
		if !info.IsDir() {
			if !strings.HasSuffix(path, ".toml") {
				return nil
			}
			var err error
			var config map[string]interface{}
			_, err = toml.DecodeFile(path, &config)
			if err != nil {
				fmt.Println("aaf: ", path, err)
				log.Logger().Fatal(nil, err)
			}
			conf.copy(strings.TrimSuffix(info.Name(), ".toml"), config)
		} else {
			return filepath.Walk(info.Name(), conf.walk)
		}
	}
	return nil
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
				conf.walk(dir+"/"+fi.Name(), fi, nil)
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

func (conf *Config) assignment(val, obj interface{}) error {
	data, _ := json.Marshal(val)
	return json.Unmarshal(data, obj)
}
