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
	"github.com/haproxytech/kubernetes-controller/controller/config"
	opt "github.com/haproxytech/kubernetes-controller/controller/options"
	"github.com/phuslu/log"
)

func main() {
	fmt.Println(string(version.Info)) //nolint
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	controllerConfig, err := createControllerPodConfig()
	if err != nil {
		panic(fmt.Errorf("error creating controller pod config: %w", err))
	}
	// Values to get from flags
	// to implement:  flags
	metricsConfig := config.MetricsConfig{
		Port:    6062,
		Enabled: false,
		Secure:  false,
	}
	gatewayClass := "haproxy"
	leaderElectionLockName := "kubernetes-controller-leader-election-lock"
	leaderElectionConfig := config.LeaderElectionConfig{
		Enabled:  true,
		LockName: leaderElectionLockName,
		Identity: controllerConfig.Name,
	}
	gatewayControllerName := "haproxy-ingress.github.io/gateway-controller"

	controller, err := controller.New(
		opt.ControllerPodConfig(controllerConfig),
		opt.GatewayClass(gatewayClass),
		opt.MetricsConfig(metricsConfig),
		opt.LeaderElectionConfig(leaderElectionConfig),
		opt.ControllerName(gatewayControllerName),
		opt.Logging(log.InfoLevel),
		opt.RLogging())
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	go controller.Run(ctx, &wg)

	<-ctx.Done()
	controller.Configuration.Logger.Info().Msg("shutting down")
	wg.Wait()
}

func createControllerPodConfig() (config.ControllerPodConfig, error) {
	podIP, err := getValueFromEnv("POD_IP")
	if err != nil {
		return config.ControllerPodConfig{}, err
	}

	ns, err := getValueFromEnv("POD_NAMESPACE")
	if err != nil {
		return config.ControllerPodConfig{}, err
	}

	name, err := getValueFromEnv("POD_NAME")
	if err != nil {
		return config.ControllerPodConfig{}, err
	}

	c := config.ControllerPodConfig{
		PodIP:     podIP,
		Namespace: ns,
		Name:      name,
	}

	return c, nil
}

func getValueFromEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("environment variable %s not set", key)
	}

	return val, nil
}
