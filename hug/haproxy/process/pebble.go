package process

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/haproxytech/client-native/v6/runtime"
	"github.com/haproxytech/client-native/v6/runtime/options"
	hapi "github.com/haproxytech/haproxy-unified-gateway/hug/haproxy/api"
	"github.com/haproxytech/haproxy-unified-gateway/hug/haproxy/params"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/logging"
)

type pebbleControl struct {
	API               hapi.HAProxyClient
	masterSocket      runtime.Runtime
	logger            *slog.Logger
	Params            params.Params
	masterSocketValid bool
}

func newpebbleControl(api hapi.HAProxyClient, param params.Params, logger *slog.Logger) *pebbleControl {
	sc := pebbleControl{
		API:    api,
		Params: param,
		logger: logger,
	}

	masterSocket, err := runtime.New(context.Background(), options.MasterSocket(MASTER_SOCKET_PATH), options.AllowDelayedStart(time.Minute, time.Second))
	if err != nil {
		sc.logger.LogAttrs(context.Background(), slog.LevelError,
			"failed to initialize master socket",
			logging.LogAttrError(err))
		return &sc
	}
	sc.masterSocketValid = true
	sc.masterSocket = masterSocket

	return &sc
}

func (c *pebbleControl) Service(action string) (string, error) {
	if c.Params.Test {
		c.logger.LogAttrs(context.Background(), slog.LevelError,
			"HAProxy would be %sed now")
		return "", nil
	}
	var cmd *exec.Cmd

	switch action { //revive:disable:identical-switch-branches
	case "start":
		// no need to start it is up already (s6)
		return "", nil
	case "stop":
		// no need to stop it (s6)
		return "", nil
	case "reload":
		if c.masterSocketValid {
			msg, err := c.masterSocket.Reload()
			if err == nil {
				c.logger.LogAttrs(context.Background(), slog.LevelDebug, msg)
				return msg, nil
			}
			c.logger.LogAttrs(context.Background(), slog.LevelError,
				"failed to reload",
				logging.LogAttrError(err))
		}

		cmd = exec.Command("pebble", "restart", "haproxy")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return "", cmd.Run()
	default:
		return "", fmt.Errorf("unknown command '%s'", action)
	}
}

func (*pebbleControl) UseAuxFile(_ bool) {
	// do nothing we always have it
}

func (c *pebbleControl) SetAPI(api hapi.HAProxyClient) {
	c.API = api
}
