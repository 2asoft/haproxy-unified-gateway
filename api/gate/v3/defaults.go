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

// Defaults is a specification for a Defaults resource
type Defaults struct {
	Spec              DefaultsSpec `json:"spec"`
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

// DefaultsSpec defines the desired state of Defaults
type DefaultsSpec struct {
	// +kubebuilder:validation:Enum=override;append
	MergeStrategy   string `json:"merge_strategy"`
	models.Defaults `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DefaultsList is a list of Global resources
type DefaultsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Defaults `json:"items"`
}

// MarshalJSON implements json.Marshaler for GlobalSpec.
// This is needed because the embedded models.Global has its own MarshalJSON,
// which would be promoted and would omit the merge_strategy field.
func (s DefaultsSpec) MarshalJSON() ([]byte, error) {
	defaultsJSON, err := s.Defaults.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(defaultsJSON, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = make(map[string]json.RawMessage)
	}
	m["merge_strategy"], err = json.Marshal(s.MergeStrategy)
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

// UnmarshalJSON implements json.Unmarshaler for GlobalSpec.
// This is needed because the embedded models.Global has its own UnmarshalJSON,
// which would be promoted and would swallow the merge_strategy field.
func (s *DefaultsSpec) UnmarshalJSON(data []byte) error {
	if err := s.Defaults.UnmarshalJSON(data); err != nil {
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
func (s *DefaultsSpec) DeepCopyInto(out *DefaultsSpec) {
	b, err := s.Defaults.MarshalBinary()
	if err == nil {
		_ = out.Defaults.UnmarshalBinary(b)
	}
	out.MergeStrategy = s.MergeStrategy
}
