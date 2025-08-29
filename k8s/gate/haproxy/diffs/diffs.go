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
package diffs

import (
	"fmt"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/structured"
)

type HaproxyConfDiffs struct {
	Created    structured.Structured
	Updated    structured.Structured
	Deleted    structured.Structured
	ReloadNeed bool
}

func (c HaproxyConfDiffs) IsEmpty() bool {
	return c.Created.IsEmpty() && c.Updated.IsEmpty() && c.Deleted.IsEmpty()
}

func (c HaproxyConfDiffs) Stats() string {
	return fmt.Sprintf("Created/Updated/Delete FE:[%d/%d/%d] BE[%d/%d/%d]",
		len(c.Created.Frontends), len(c.Updated.Frontends), len(c.Deleted.Frontends),
		len(c.Created.Backends), len(c.Updated.Backends), len(c.Deleted.Backends),
	)
}
