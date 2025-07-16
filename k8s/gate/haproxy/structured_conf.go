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
	"fmt"

	"github.com/haproxytech/client-native/v6/models"
)

type HaproxyCfgDiffs struct {
	Created HaproxyCfg
	Updated HaproxyCfg
	Deleted HaproxyCfg
}

type HaproxyCfg struct {
	Frontends map[string]*models.Frontend
	Backends  map[string]*models.Backend
}

const (
	MetatDataKey = "k8s-unified-ctl"
)

type MetaData map[string]any

func NewHaproxyCfg() HaproxyCfg {
	return HaproxyCfg{
		Frontends: make(map[string]*models.Frontend),
		Backends:  make(map[string]*models.Backend),
	}
}

type TemplateData struct {
	// revive:disable:var-naming
	GATEWAY_NAMESPACE string
	GATEWAY_NAME      string
	LISTENER_NAME     string
	LINK_ID           string
	// revive:enable:var-naming
}

func (c HaproxyCfgDiffs) IsEmpty() bool {
	return c.Created.IsEmpty() && c.Updated.IsEmpty() && c.Deleted.IsEmpty()
}

func (c HaproxyCfg) IsEmpty() bool {
	return len(c.Frontends) == 0 && len(c.Backends) == 0
}

func (c HaproxyCfgDiffs) Stats() string {
	return fmt.Sprintf("Created[FE:%d/BE:%d] Modified[FE:%d/BE:%d] Deleted[FE:%d/BE:%d]",
		len(c.Created.Frontends), len(c.Created.Backends),
		len(c.Updated.Frontends), len(c.Updated.Backends),
		len(c.Deleted.Frontends), len(c.Deleted.Backends),
	)
}

// Maps
// Certs
// Runtime
