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
package jobs

import (
	"fmt"

	"github.com/haproxytech/haproxy-unified-gateway/api/definition"
)

// CRDInstall installs or updates the HUG CRDs in the cluster.
// If external is true, it uses the kubeconfig from ~/.kube/config,
// otherwise it uses the in-cluster config.
func CRDInstall(external bool) error {
	fmt.Print(hugCRDInstaller)
	return definition.CRDRefresh(external)
}

const hugCRDInstaller = `
  _   _ _   _  ____    ____ ____  ____    ___           _        _ _
 | | | | | | |/ ___|  / ___|  _ \|  _ \  |_ _|_ __  ___| |_ __ _| | | ___ _ __
 | |_| | | | | |  _  | |   | |_) | | | |  | || '_ \/ __| __/ _` + "`" + ` | | |/ _ \ '__|
 |  _  | |_| | |_| | | |___|  _ <| |_| |  | || | | \__ \ || (_| | | |  __/ |
 |_| |_|\___/ \____|  \____|_| \_\____/  |___|_| |_|___/\__\__,_|_|_|\___|_|
`
