// Copyright 2019 HAProxy Technologies
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package v3

import (
	"encoding/json"

	"github.com/haproxytech/client-native/v6/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:metadata:annotations="client-native.haproxy.org/version=v6.2.5"

// Global is a specification for a Global resource
type Global struct {
	Spec              GlobalSpec `json:"spec"`
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

// GlobalSpec defines the desired state of Global
//
// Note: the following fields from models.Global are intentionally excluded from this CRD
// and cannot be set (they are managed by the controller or not applicable in Kubernetes):
//   - daemon
//   - localpeer
//   - master-worker
//   - pidfile
//   - stats_timeout
//   - default_path
//   - tune_lua_options.bool_sample_conversion
//   - lua_options.load_per_thread
//
// The removal of those fields is done after the CRD is generated from models.Global
type GlobalSpec struct {
	// +kubebuilder:validation:Enum=override;append
	MergeStrategy string `json:"merge_strategy"`
	models.Global `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalList is a list of Global resources
type GlobalList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Global `json:"items"`
}

// UnmarshalJSON implements json.Unmarshaler for GlobalSpec.
// This is needed because the embedded models.Global has its own UnmarshalJSON,
// which would be promoted and would swallow the merge_strategy field.
func (s *GlobalSpec) UnmarshalJSON(data []byte) error {
	if err := s.Global.UnmarshalJSON(data); err != nil {
		return err
	}
	var shadow struct {
		MergeStrategy string `json:"merge_strategy"`
	}
	if err := json.Unmarshal(data, &shadow); err != nil {
		return err
	}
	s.MergeStrategy = shadow.MergeStrategy
	return nil
}

// DeepCopyInto deepcopying  the receiver into out. in must be non-nil.
func (s *GlobalSpec) DeepCopyInto(out *GlobalSpec) {
	b, err := s.Global.MarshalBinary()
	if err == nil {
		_ = out.Global.UnmarshalBinary(b)
	}
	out.MergeStrategy = s.MergeStrategy
}
