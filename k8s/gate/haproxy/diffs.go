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

import "fmt"

type HaproxyConfDiffs struct {
	Created Structured
	Updated Structured
	Deleted Structured
}

func (c HaproxyConfDiffs) IsEmpty() bool {
	return c.Created.IsEmpty() && c.Updated.IsEmpty() && c.Deleted.IsEmpty()
}

func (c HaproxyConfDiffs) Stats() string {
	return fmt.Sprintf("Created[FE:%d/BE:%d] Updated[FE:%d/BE:%d] Deleted[FE:%d/BE:%d]",
		len(c.Created.Frontends), len(c.Created.Backends),
		len(c.Updated.Frontends), len(c.Updated.Backends),
		len(c.Deleted.Frontends), len(c.Deleted.Backends),
	)
}
