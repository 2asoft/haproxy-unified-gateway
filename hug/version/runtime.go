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
package version

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
)

var (
	Repo    = ""
	Version = "dev"
	// CommitDate = ""
)

var ErrBuildDataNotReadable = errors.New("not able to read build data")

func Set() error {
	buildinfo, ok := debug.ReadBuildInfo()
	if !ok {
		return ErrBuildDataNotReadable
	}
	Repo = buildinfo.Main.Path
	// CommitDate = get(buildinfo, "vcs.time")

	Version = strings.Replace(buildinfo.Main.Version, "(devel)", "dev", 1)

	return nil
}

// func get(buildInfo *debug.BuildInfo, key string) string {
// 	for _, setting := range buildInfo.Settings {
// 		if setting.Key == key {
// 			return setting.Value
// 		}
// 	}
// 	return ""
// }

// PrintVersion prints version information to standard output
func PrintVersion() {
	fmt.Println("HAProxy unified gateway", Version)
	fmt.Println("built-from", Repo)
	// if CommitDate != "" {
	// 	fmt.Println("commit-date", CommitDate)
	// }
}
