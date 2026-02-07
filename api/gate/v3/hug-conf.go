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
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

// if we plan to change structure of this CRD in future versions, we need to
// update the hug/version annotation below and write hash in api/gate/v3/versions.yml

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:metadata:annotations="conf.hug/version=v0.7.0"

// HugConf is a specification for a the controller related configuration
type HugConf struct {
	metav1.TypeMeta   `json:",inline"`
	Spec              ControllerConfSpec `json:"spec"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

type (
	// +kubebuilder:validation:Enum=k8s;gate;status;haproxycfg;app;batch;reloadmgr;certs-storage;maps-storage;hugservice
	Category string
	// +kubebuilder:validation:Enum=Debug;Info;Warn;Error;None
	// +kubebuilder:validation:Required
	Level string
)

type CategoryLevel struct {
	Category Category `json:"category"`
	Level    Level    `json:"level"`
}

type Logging struct {
	DefaultLevel Level `json:"defaultLevel,omitempty"`
	// CategoryLevelList is a list of categories and their levels
	// if a category is not present, the default level is used
	CategoryLevelList []CategoryLevel `json:"categoryLevelList,omitempty"`
}

type CRReference struct {
	// Group is the group of the referent.
	Group *v1.Group `json:"group,omitempty"`

	// Kind is the kind of the referent.
	Kind *v1.Kind `json:"kind,omitempty"`

	// Namespace is the namespace of the referent.
	Namespace *v1.Namespace `json:"namespace,omitempty"`

	// Name is the name of the referent.
	Name v1.ObjectName `json:"name"`
}

type CaptureRequestHeader struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Minimum=1
	Length int64 `json:"length,omitempty"`
}

type HaproxyGlobal struct {
	// +kubebuilder:validation:Minimum=1
	LogLineLength *int64 `json:"logLineLength,omitempty"`
	// +kubebuilder:validation:Minimum=1
	HTTPLogURILen *int64 `json:"httpLogUriLen,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="!self.logRequestHeaderNames || size(self.logFormat) > 0",message="logFormat is required when logRequestHeaderNames is enabled"
type HaproxyDefaults struct {
	LogFormat             string                 `json:"logFormat,omitempty"`
	LogRequestHeaderNames bool                   `json:"logRequestHeaderNames,omitempty"`
	CaptureRequestHeaders []CaptureRequestHeader `json:"captureRequestHeaders,omitempty"`
}
type ControllerConfSpec struct {
	GlobalRef       *CRReference     `json:"globalRef,omitempty"`
	DefaultsRef     *CRReference     `json:"defaultsRef,omitempty"`
	Logging         Logging          `json:"logging"`
	HaproxyDefaults *HaproxyDefaults `json:"haproxyDefaults,omitempty"`
	HaproxyGlobal   *HaproxyGlobal   `json:"haproxyGlobal,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HugConfList is a list of HugConf resources
type HugConfList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []HugConf `json:"items"`
}
