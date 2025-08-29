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
	"log/slog"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/certificate"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
)

type ControllerStore struct {
	ClusterStore      *store.ClusterStore
	GateTree          *GateTree
	UnmanagedGateTree *GateTree
	ReferencedObjects *ReferencedObjects
	// A Map of installed GwApi CRDs versions
	InstalledGwAPIVersions *InstalledVersions
	Logger                 *slog.Logger
	ExtractGVK             utils.ExtractGVK
	CertificateUpdates     *CertificateUpdates
	CrtListUpdates         *CrtListUpdates
	Certificates           map[string]certificate.CertificateData // cert file name
	CrtLists               map[string]certificate.CrtListData     // crt-list file name
}

type CertificateUpdates struct {
	Created map[string]certificate.CertificateData
	Updated map[string]certificate.CertificateData
	Deleted map[string]certificate.CertificateData
}

func (b *ControllerStore) CleanInstalledVersionsUpdates() {
	if b.InstalledGwAPIVersions.Updated != nil {
		*b.InstalledGwAPIVersions.Updated = false
	}
}

func (b *ControllerStore) addCreatedCertificate(certData certificate.CertificateData) {
	b.CertificateUpdates.Created[certData.MapKey()] = certData
}

func (b *ControllerStore) addUpdatedCertificate(certData certificate.CertificateData) {
	b.CertificateUpdates.Updated[certData.MapKey()] = certData
}

func (b *ControllerStore) addDeletedCertificate(certData certificate.CertificateData) {
	b.CertificateUpdates.Deleted[certData.MapKey()] = certData
}

func (b *ControllerStore) ResetCertificateUpdates() {
	b.CertificateUpdates.Created = make(map[string]certificate.CertificateData)
	b.CertificateUpdates.Updated = make(map[string]certificate.CertificateData)
	b.CertificateUpdates.Deleted = make(map[string]certificate.CertificateData)
}

type CrtListUpdates struct {
	Created map[string]certificate.CrtListData
	Updated map[string]certificate.CrtListData
	Deleted map[string]certificate.CrtListData
}

func (b *ControllerStore) addCreatedCrtList(crtListData certificate.CrtListData) {
	b.CrtListUpdates.Created[crtListData.MapKey()] = crtListData
}

func (b *ControllerStore) addUpdatedCrtList(crtListData certificate.CrtListData) {
	b.CrtListUpdates.Updated[crtListData.MapKey()] = crtListData
}

func (b *ControllerStore) addDeletedCrtList(crtListData certificate.CrtListData) {
	b.CrtListUpdates.Deleted[crtListData.MapKey()] = crtListData
}

func (b *ControllerStore) ResetCrtListUpdates() {
	b.CrtListUpdates.Created = make(map[string]certificate.CrtListData)
	b.CrtListUpdates.Updated = make(map[string]certificate.CrtListData)
	b.CrtListUpdates.Deleted = make(map[string]certificate.CrtListData)
}
