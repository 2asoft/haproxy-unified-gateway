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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// HaproxyGateCtrlCfg is a specification for a the controller related configuration
type HaproxyGateCtrlCfg struct {
	metav1.TypeMeta   `json:",inline"`
	Spec              ControllerConfSpec `json:"spec"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

// +kubebuilder:validation:Enum=all;k8s;gate;status
type Category string

type Logging struct {
	// +kubebuilder:validation:Enum=Debug;Info;Warn;Error
	Level      string     `json:"level"`
	Categories []Category `json:"categories"`
}
type ControllerConfSpec struct {
	Logging Logging `json:"logging"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HaproxyGateCtrlCfgList is a list of HaproxyGateCtrlrConf resources
type HaproxyGateCtrlCfgList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []HaproxyGateCtrlCfg `json:"items"`
}
