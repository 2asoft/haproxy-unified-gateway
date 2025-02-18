package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/haproxytech/kubernetes-controller/cmd/controller/version"
	"github.com/haproxytech/kubernetes-controller/controller"
	opt "github.com/haproxytech/kubernetes-controller/controller/options"
)

func main() {
	fmt.Println(string(version.Info)) //nolint
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	controller, err := controller.New(opt.GatewayClass("haproxy"), opt.Logging())
	if err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	go controller.Run(ctx, &wg)

	<-ctx.Done()
	controller.Configuration.Logger.Info().Msg("shutting down")
	wg.Wait()
	controller.Configuration.Logger.Info().Msg("done")
}
