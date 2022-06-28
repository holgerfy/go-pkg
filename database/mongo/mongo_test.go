package mongo

import (
	"fmt"
	"github.com/holgerfy/go-pkg/config"
	"github.com/holgerfy/go-pkg/funcs"
	"github.com/holgerfy/go-pkg/log"
	"github.com/holgerfy/go-pkg/redis"
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	log.Start()
	config.LoadConfig([]string{funcs.GetRoot()})
	Start()
	redis.Start()

	//mongo.Client.Database("tmm").Collection("")
	num, err := Database("tmm_im").SetTable("user_info").Count()
	fmt.Println(num, err)

	if redis.Lock("test", 3) {
		fmt.Println("lock successfully")
	} else {
		fmt.Println("lock failed")
	}
	redis.Client.Set("test", "wew", time.Second*4)
}
