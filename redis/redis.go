package redis

import (
	"crypto/tls"
	"github.com/go-redis/redis"
	"github.com/holgerfy/go-pkg/config"
	"time"
)

var (
	Client *redis.Client
	conf   struct {
		Addr         string `toml:"addr"`
		Password     string `toml:"password"`
		Db           int    `toml:"dao"`
		PoolSize     int    `toml:"pool_size"`
		MinIdleConns int    `toml:"min_idle_conns"`
		IsEnableTls  int    `toml:"is_enable_tls"`
	}
	NilErr = redis.Nil
)

func Start() {
	err := config.GetInstance().Bind("db", "redis", &conf)
	if err == config.ErrNodeNotExists {
		return
	}
	opt := &redis.Options{
		Addr:         conf.Addr,
		Password:     conf.Password,
		DB:           conf.Db,
		PoolSize:     conf.PoolSize,
		MinIdleConns: conf.MinIdleConns,
	}
	if conf.Password != "" {
		opt.Password = conf.Password
	}
	if conf.IsEnableTls == 1 {
		opt.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		}
	}
	Client = redis.NewClient(opt)
}

func CacheGet(key string, expiration time.Duration, f func() string) string {
	cmd := Client.Get(key)
	var val string
	result, _ := cmd.Result()
	if len(result) == 0 {
		Client.Set(key, f(), expiration)
		return val
	}
	return result
}

func Lock(key string, expire int) bool {
	lockName := "lock:" + key
	lockTimeOut := time.Duration(expire) * time.Second
	if ok, err := Client.SetNX(lockName, 1, lockTimeOut).Result(); err != nil && err != NilErr {
		return false
	} else if ok {
		return true
	} else if Client.TTL(lockName).Val() == -1 { // -2: expire ；-1：no expire；
		Client.Expire(lockName, lockTimeOut)
	}
	return false
}

func Unlock(key string) bool {
	lockName := "lock:" + key
	num, err := Client.Del(lockName).Result()
	if err != nil && err == redis.Nil {
		return true
	} else if err == nil && num > 0 {
		return true
	}
	return false
}
