package process

import (
	"bufio"
	"log/slog"
	"os"
	"strconv"
	"syscall"

	hapi "github.com/haproxytech/kubernetes-controller/hug/haproxy/api"
	"github.com/haproxytech/kubernetes-controller/hug/haproxy/params"
)

// MUST be the same as in fs/etc/s6-overlay/s6-rc.d/haproxy/run
const MASTER_SOCKET_PATH = "/var/run/haproxy-master.sock" // revive:disable:var-naming

type Process interface {
	Service(action string) (err error)
	UseAuxFile(useAuxFile bool)
	SetAPI(api hapi.HAProxyClient)
}

func New(param params.Params, api hapi.HAProxyClient, logger *slog.Logger) (p Process) { //nolint:ireturn
	switch {
	case param.UseWiths6Overlay:
		p = newS6Control(api, param, logger)
	default:
		p = newDirectControl(api, param, logger)
		if _, err := os.Stat(param.AuxDir); err == nil {
			p.UseAuxFile(true)
		}
	}
	return p
}

// Return HAProxy master process if it exists.
func haproxyProcess(pidFile string) (*os.Process, error) {
	file, err := os.Open(pidFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	pid, err := strconv.Atoi(scanner.Text())
	if err != nil {
		return nil, err
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return nil, err
	}
	err = process.Signal(syscall.Signal(0))
	return process, err
}
