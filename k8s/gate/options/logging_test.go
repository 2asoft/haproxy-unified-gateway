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
package opt

import (
	"log/slog"
	"testing"

	controller "github.com/haproxytech/kubernetes-controller/k8s/gate"
)

func TestLogging(t *testing.T) {
	_, err := controller.New(Logging(slog.LevelDebug))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoggingEmpty(t *testing.T) {
	_, err := controller.New()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
