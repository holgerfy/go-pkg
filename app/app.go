package app

import (
	"os"
	"path/filepath"
	"sync"
)

const (
	EnvModelRelease = "release"
	EnvModelDebug   = "debug"
	EnvModelDev     = "dev"
	EnvModelLocal   = "local"
)

var services = &Services{services: make(map[string]interface{})}

type Services struct {
	lock     sync.Mutex
	services map[string]interface{}
}

func (service *Services) register(name string, se interface{}) {
	service.lock.Lock()
	defer service.lock.Unlock()

	service.services[name] = se
}

func (service *Services) get(name string) interface{} {
	if val, ok := service.services[name]; ok {
		return val
	}
	return nil
}

func Root() string {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	return dir
}

func Name() string {
	stat, _ := os.Stat(os.Args[0])
	return stat.Name()
}

func Register(name string, service interface{}) interface{} {
	services.register(name, service)
	return service
}

func Get(name string) interface{} {
	return services.get(name)
}
