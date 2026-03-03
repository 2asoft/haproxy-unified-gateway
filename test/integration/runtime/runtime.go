// Copyright 2019 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package truntime

import (
	"bufio"
	"bytes"
	"net"
	"regexp"
	"strconv"
	"strings"
)

//revive:disable:var-naming
const (
	STATS_PORT = 31024
)

//revive:enable:var-naming

type Runtime struct {
	SocketPath string
}

func NewRuntime(socketPath string) Runtime {
	return Runtime{
		SocketPath: socketPath,
	}
}

type GlobalHAProxyInfo struct {
	Pid     string
	Maxconn string
	Uptime  string
}

func (*Runtime) GetHAProxyMapCount(socketPath string, mapName string) (count int, err error) {
	var result []byte
	result, err = runtimeCommand(socketPath, "show map")
	if err != nil {
		return count, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(result))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, mapName+".map") {
			r := regexp.MustCompile("entry_cnt=[0-9]*")
			match := r.FindString(line)
			nbr := strings.Split(match, "=")[1]
			count, err = strconv.Atoi(nbr)
			break
		}
	}
	return count, err
}

func GetGlobalHAProxyInfo(socketPath string) (info GlobalHAProxyInfo, err error) {
	var result []byte
	result, err = runtimeCommand(socketPath, "show info")
	if err != nil {
		return info, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(result))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "Maxconn:"):
			info.Maxconn = strings.Split(line, ": ")[1]
		case strings.HasPrefix(line, "Uptime:"):
			info.Uptime = strings.Split(line, ": ")[1]
		case strings.HasPrefix(line, "Pid:"):
			info.Pid = strings.Split(line, ": ")[1]
		}
	}
	return info, err
}

func runtimeCommand(socketPath string, command string) (result []byte, err error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_, err = conn.Write([]byte(command + "\n"))
	if err != nil {
		return result, err
	}
	result = make([]byte, 16384)
	_, err = conn.Read(result)
	if err != nil {
		return []byte{}, err
	}
	err = conn.Close()
	return result, err
}

func GetServers(socketPath string, backend string) ([]string, error) {
	var result []byte
	command := "show stat " + backend + " 4 -1"
	result, err := runtimeCommand(socketPath, command)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(result))
	servers := make([]string, 0)
	i := 0
	for scanner.Scan() {
		line := scanner.Text()
		if i == 0 {
			// Skip the first line which contains column headers
			i++
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			servers = append(servers, parts[1])
		}
	}
	return servers, nil
}
