// Copyright 2024 HAProxy Technologies
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// if we plan to change structure of this CRD in future versions, we need to
// update the hug/version annotation below and write hash in api/gate/v3/versions.yml

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:metadata:annotations="gate.hug/version=v0.7.0"

// HugGate is a specification for a HugGate resource
type HugGate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GateSpec `json:"spec"`
}

type GateSpec struct {
	MWorkerMaxReload int `json:"mworkerMaxReload"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HugGateList is a list of HugGate resources
type HugGateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []HugGate `json:"items"`
}
