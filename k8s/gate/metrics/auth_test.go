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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

func TestWithBasicAuth_ValidCredentials(t *testing.T) {
	provider := WithBasicAuth("admin", "secret")
	filter, err := provider(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error creating filter: %v", err)
	}
	handler, err := filter(logr.Discard(), okHandler())
	if err != nil {
		t.Fatalf("unexpected error wrapping handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.SetBasicAuth("admin", "secret")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if rr.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got %q", rr.Body.String())
	}
}

func TestWithBasicAuth_NoCredentials(t *testing.T) {
	provider := WithBasicAuth("admin", "secret")
	filter, err := provider(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error creating filter: %v", err)
	}
	handler, err := filter(logr.Discard(), okHandler())
	if err != nil {
		t.Fatalf("unexpected error wrapping handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	if rr.Header().Get("WWW-Authenticate") != `Basic realm="metrics"` {
		t.Errorf("expected WWW-Authenticate header, got %q", rr.Header().Get("WWW-Authenticate"))
	}
}

func TestWithBasicAuth_WrongUsername(t *testing.T) {
	provider := WithBasicAuth("admin", "secret")
	filter, _ := provider(nil, nil)
	handler, _ := filter(logr.Discard(), okHandler())

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.SetBasicAuth("wrong", "secret")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestWithBasicAuth_WrongPassword(t *testing.T) {
	provider := WithBasicAuth("admin", "secret")
	filter, _ := provider(nil, nil)
	handler, _ := filter(logr.Discard(), okHandler())

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.SetBasicAuth("admin", "wrong")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestWithBasicAuth_EmptyCredentials(t *testing.T) {
	provider := WithBasicAuth("", "")
	filter, _ := provider(nil, nil)
	handler, _ := filter(logr.Discard(), okHandler())

	// Empty creds configured, empty creds sent
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.SetBasicAuth("", "")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}
