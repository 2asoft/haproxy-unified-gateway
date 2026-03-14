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
package metrics

import (
	"crypto/subtle"
	"net/http"

	"github.com/go-logr/logr"
	"k8s.io/client-go/rest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// WithBasicAuth returns a FilterProvider that enforces HTTP Basic Authentication
// on the metrics endpoint using the provided username and password.
func WithBasicAuth(username, password string) func(c *rest.Config, httpClient *http.Client) (metricsserver.Filter, error) {
	return func(_ *rest.Config, _ *http.Client) (metricsserver.Filter, error) {
		return func(_ logr.Logger, handler http.Handler) (http.Handler, error) {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				user, pass, ok := r.BasicAuth()
				if !ok ||
					subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 ||
					subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
					w.Header().Set("WWW-Authenticate", `Basic realm="metrics"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				handler.ServeHTTP(w, r)
			}), nil
		}, nil
	}
}
