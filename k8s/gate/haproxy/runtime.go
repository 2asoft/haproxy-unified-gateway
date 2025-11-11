// Copyright 2025 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package haproxy

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/logging"
)

// BufferSize is the default value of HAproxy tune.bufsize. Not recommended to change it
// Map payload or socket data cannot be bigger than tune.bufsize
const BufferSize = 16000

type RuntimeServerStateData struct {
	BackendName string
	ServerName  string
	IP          string
	State       string
	Port        int
}

func (b *HaproxyConfMgrImpl) RuntimeSetServerAddrAndState(servers []RuntimeServerStateData) error {
	if len(servers) == 0 {
		return nil
	}

	backendNameSize := len(servers[0].BackendName)
	oneServerCommandSize := 75 + 2*backendNameSize
	size := oneServerCommandSize * len(servers)
	if size > BufferSize {
		size = BufferSize
	}

	var sb strings.Builder
	sb.Grow(size)
	var cmdBuilder strings.Builder
	cmdBuilder.Grow(oneServerCommandSize)
	for _, server := range servers {
		// if new commands are added recalculate oneServerCommandSize
		cmdBuilder.WriteString("set server ")
		cmdBuilder.WriteString(server.BackendName)
		cmdBuilder.WriteString("/")
		cmdBuilder.WriteString(server.ServerName)
		cmdBuilder.WriteString(" addr ")
		cmdBuilder.WriteString(server.IP)
		if server.Port > 0 {
			cmdBuilder.WriteString(" port ")
			cmdBuilder.WriteString(strconv.Itoa(server.Port))
		}
		cmdBuilder.WriteString(";set server ")
		cmdBuilder.WriteString(server.BackendName)
		cmdBuilder.WriteString("/")
		cmdBuilder.WriteString(server.ServerName)
		cmdBuilder.WriteString(" state ")
		cmdBuilder.WriteString(server.State)
		cmdBuilder.WriteString(";")
		// if new commands are added recalculate oneServerCommandSize

		if sb.Len()+cmdBuilder.Len() >= BufferSize {
			err := b.runRaw(sb, server.BackendName)
			if err != nil {
				return err
			}
			sb.Reset()
			sb.Grow(size)
		}
		sb.WriteString(cmdBuilder.String())
		cmdBuilder.Reset()
		cmdBuilder.Grow(oneServerCommandSize)
	}
	if sb.Len() > 0 {
		err := b.runRaw(sb, servers[0].BackendName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *HaproxyConfMgrImpl) runRaw(sb strings.Builder, backendName string) error {
	runtime := b.haproxyClient.RuntimeClient()
	result, err := runtime.ExecuteRaw(sb.String())
	if err != nil {
		return err
	}
	if len(result) > 5 {
		switch result[0:4] {
		case "[3]:", "[2]:", "[1]:", "[0]:":
			b.logger.LogAttrs(context.Background(), slog.LevelError, "runtime update failed for "+backendName, logging.LogAttrBackendName(backendName))
			err := errors.New("runtime update failed for " + backendName)
			return err
		}
	}
	return nil
}

func (b *HaproxyConfMgrImpl) RuntimeDeleteServer(backendName, serverName string) error {
	runtime := b.haproxyClient.RuntimeClient()
	err := runtime.DeleteServer(backendName, serverName)
	if err != nil {
		return err
	}
	return nil
}
