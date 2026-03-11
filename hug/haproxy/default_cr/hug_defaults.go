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
package defaultcrs

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/haproxytech/client-native/v6/models"
	k8syaml "sigs.k8s.io/yaml"
)

//go:embed default_global.yaml
var defaultGlobalYAML []byte

type defaultGlobalParams struct {
	RuntimeSocket string
	PIDFile       string
}

func DefaultGlobal(runtimeSocket, pidFile string) (models.Global, error) {
	tmpl, err := template.New("default_global").Parse(string(defaultGlobalYAML))
	if err != nil {
		return models.Global{}, err
	}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, defaultGlobalParams{RuntimeSocket: runtimeSocket, PIDFile: pidFile}); err != nil {
		return models.Global{}, err
	}
	var global models.Global
	if err = k8syaml.Unmarshal(buf.Bytes(), &global); err != nil {
		return models.Global{}, err
	}
	return global, nil
}
