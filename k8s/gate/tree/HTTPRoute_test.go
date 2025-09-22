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
package tree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

//revive:disable:function-length
func Test_match(t *testing.T) {
	tests := []struct {
		name             string
		routeHostname    string
		listenerHostname string
		want             bool
	}{
		{
			name:             "exact match",
			routeHostname:    "example.com",
			listenerHostname: "example.com",
			want:             true,
		},
		{
			name:             "wildcard listener, matching route",
			routeHostname:    "foo.example.com",
			listenerHostname: "*.example.com",
			want:             true,
		},
		{
			name:             "wildcard listener, non-matching route",
			routeHostname:    "example.com",
			listenerHostname: "*.example.com",
			want:             false,
		},
		{
			name:             "wildcard listener, different subdomain",
			routeHostname:    "foo.example.org",
			listenerHostname: "*.example.com",
			want:             false,
		},
		{
			name:             "wildcard route, matching listener",
			routeHostname:    "*.example.com",
			listenerHostname: "foo.example.com",
			want:             true,
		},
		{
			name:             "wildcard route, non-matching listener",
			routeHostname:    "*.example.com",
			listenerHostname: "example.com",
			want:             false,
		},
		{
			name:             "wildcard route, different subdomain",
			routeHostname:    "*.example.com",
			listenerHostname: "foo.example.org",
			want:             false,
		},
		{
			name:             "double wildcard listener",
			routeHostname:    "bar.foo.example.com",
			listenerHostname: "*.foo.example.com",
			want:             true,
		},
		{
			name:             "double wildcard route",
			routeHostname:    "*.foo.example.com",
			listenerHostname: "bar.foo.example.com",
			want:             true,
		},
		{
			name:             "no match",
			routeHostname:    "one.com",
			listenerHostname: "two.com",
			want:             false,
		},
		{
			name:             "empty strings",
			routeHostname:    "",
			listenerHostname: "",
			want:             true,
		},
		{
			name:             "empty route string",
			routeHostname:    "",
			listenerHostname: "example.com",
			want:             false,
		},
		{
			name:             "empty listener string",
			routeHostname:    "example.com",
			listenerHostname: "",
			want:             false,
		},
		{
			name:             "wildcard listener matches route TLD",
			routeHostname:    "example.com",
			listenerHostname: "*.com",
			want:             true,
		},
		{
			name:             "wildcard route matches listener TLD",
			routeHostname:    "*.com",
			listenerHostname: "example.com",
			want:             true,
		},
		{
			name:             "wildcard listener does not match partial",
			routeHostname:    "foo.bar.com",
			listenerHostname: "*.com",
			want:             true,
		},
		{
			name:             "wildcard route does not match partial",
			routeHostname:    "*.com",
			listenerHostname: "foo.bar.com",
			want:             true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, match(tt.routeHostname, tt.listenerHostname))
		})
	}
}
