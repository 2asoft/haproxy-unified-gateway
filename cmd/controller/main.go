package main

import (
	"github.com/haproxytech/kubernetes-controller/controller"
	opt "github.com/haproxytech/kubernetes-controller/controller/options"
)

func main() {
	controller, err := controller.New(opt.GatewayClass("haproxy"))
	if err != nil {
		panic(err)
	}
	controller.Run()
}
