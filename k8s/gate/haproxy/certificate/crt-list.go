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
package certificate

import (
	futils "github.com/haproxytech/kubernetes-controller/k8s/gate/fileutils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CrtListData struct {
	ImpactedFrontends map[string]struct{}           // map[frontendName]
	ImpactedListeners map[client.ObjectKey]struct{} // map[gatewayKey]}
	// Path is set only if the certificate was saved on disk (this is a gate configuration)
	// For HUG, certificates are stored on disk
	Path futils.FilePath
	// Content contains list of certificates full paths
	// Only set if the crt-list is not saved on disk
	// For HUG, Data will be empty
	Content []string
	// ReloadNeeded is managed only if the gate library did try to run a runtime command (this is a gate configuration)
	// For HUG, the gate library will try to create/update/delete certificate through runtime
	// By default, it is set to true
	// If the gate library did try to update the certificates through runtime (create/update/delete), and a reload is not needed
	// as the runtime command did succeed, then the flag is set to false
	ReloadNeeded bool
}

func NewCrtListData(path futils.FilePath, certificateFullPaths []string) CrtListData {
	return CrtListData{
		Path:         path,
		Content:      certificateFullPaths,
		ReloadNeeded: true,
	}
}

func (c CrtListData) MapKey() string {
	return c.Path.FileName
}
