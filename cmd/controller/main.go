package main

import (
	"fmt"

	"github.com/haproxytech/kubernetes-controller/cmd/controller/version"
	"github.com/haproxytech/kubernetes-controller/controller"
	opt "github.com/haproxytech/kubernetes-controller/controller/options"
)

func main() {
	fmt.Println(string(version.Info)) //nolint
	controller, err := controller.New(opt.GatewayClass("haproxy"), opt.Logging())
	if err != nil {
		panic(err)
	}
	controller.Run()
}
