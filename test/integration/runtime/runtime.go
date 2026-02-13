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
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

//revive:disable:var-naming
const (
	STATS_PORT = 31024
)

//revive:enable:var-naming

type GlobalHAProxyInfo struct {
	Pid     string
	Maxconn string
	Uptime  string
}

func GetHAProxyMapCount(mapName string) (count int, err error) {
	var result []byte
	result, err = runtimeCommand("show map")
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

func GetGlobalHAProxyInfo() (info GlobalHAProxyInfo, err error) {
	var result []byte
	result, err = runtimeCommand("show info")
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

func runtimeCommand(command string) (result []byte, err error) {
	kindURL := os.Getenv("KIND_URL")
	if kindURL == "" {
		kindURL = "127.0.0.1"
	}
	conn, err := net.Dial("tcp", net.JoinHostPort(kindURL, fmt.Sprintf("%d", STATS_PORT)))
	if err != nil {
		return result, err
	}
	_, err = conn.Write([]byte(command + "\n"))
	if err != nil {
		return result, err
	}
	result = make([]byte, 2048)
	_, err = conn.Read(result)
	if err != nil {
		return []byte{}, err
	}
	err = conn.Close()
	return result, err
}
