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
	futils "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/fileutils"
)

type CertificateData struct {
	// Path is set only if the certificate was saved on disk (this is a gate configuration)
	// For HUG, certificates are stored on disk
	Path futils.FilePath
	// Data contains the secret .pem content
	// Only set if the certificate is not saved on disk
	// For HUG, Data will be empty as the .pem is saved on disk
	Data []byte
	// ReloadNeeded is managed only if the gate library did try to run a runtime command (this is a gate configuration)
	// For HUG, the gate library will try to create/update/delete certificate through runtime
	// By default, it is set to true
	// If the gate library did try to update the certificates through runtime (create/update/delete), and a reload is not needed
	// as the runtime command did succeed, then the flag is set to false
	ReloadNeeded bool
}

func NewCertificateData(path futils.FilePath, data []byte) CertificateData {
	return CertificateData{
		Path:         path,
		Data:         data,
		ReloadNeeded: true,
	}
}

func (c CertificateData) MapKey() string {
	return c.Path.FileName
}
