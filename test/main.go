package main

import (
	"context"
	"fmt"
	"github.com/holgerfy/go-pkg/config"
	"github.com/holgerfy/go-pkg/funcs"
	"github.com/holgerfy/go-pkg/log"
	"github.com/holgerfy/go-pkg/unique"
	"net/http"
)

func main() {
	config.LoadConfig([]string{funcs.GetRoot() + "/test-go-pkg/"})
	log.Start()
	http.HandleFunc("/test", test)
	err := http.ListenAndServe(":8888", nil)
	if err != nil {
		fmt.Println(err)
	}
}

func test(w http.ResponseWriter, r *http.Request) {
	uuid := unique.Uuid()
	ctx := log.WithFields(context.Background(), map[string]string{"req-id": uuid})
	log.Logger().Info(ctx, "test--: ", uuid)
	w.Write([]byte("hello "))
}
