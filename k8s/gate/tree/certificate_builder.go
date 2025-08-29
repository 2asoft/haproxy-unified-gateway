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
	"context"
	"log/slog"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/certificate"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/storage"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	objtypes "github.com/haproxytech/kubernetes-controller/k8s/gate/object-types"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ Builder = &CertificateBuilderImpl{}

type CertificateBuilderImpl struct {
	ControllerStore
	certStorage storage.CertificateStorage
	// storeCertificatesOnDisk is a flag that indicates to the gate library to store certificates on disk
	storeCertificateOnDisk bool
	// try to perform runtime update of haproxy using runtime socket
	runtimeUpdateHaproxy bool
}

func NewCertificateBuilder(controllerStore ControllerStore, storeCertOnDisk, runtimeUpdate bool, certStorage storage.CertificateStorage) Builder {
	return &CertificateBuilderImpl{
		ControllerStore:        controllerStore,
		storeCertificateOnDisk: storeCertOnDisk,
		certStorage:            certStorage,
		runtimeUpdateHaproxy:   runtimeUpdate,
	}
}

// --------------------
// GateTree Updates
// --------------------

// --------------------

func (b *CertificateBuilderImpl) ComputeTreeUpdates() {
	b.computeCertificateDiffs()
	if b.storeCertificateOnDisk {
		b.ensureCertificatesStorage()
		err := b.certStorage.DeleteEmptyCertsDir()
		if err != nil {
			b.Logger.LogAttrs(context.Background(), slog.LevelError, "error deleting empty cert dirs",
				logging.LogAttrError(err))
		}
	}
	if b.runtimeUpdateHaproxy {
		b.runtimeUpdates()
	}
}

func (*CertificateBuilderImpl) CleanTreeUpdates() {
}

// -----------------------------------------------

func (b *CertificateBuilderImpl) computeCertificateDiffs() {
	// Current secrets referenced by a Gateway
	currentSecretsReferenced := b.ControllerStore.ReferencedObjects.ReferencedSecrets.AllReferenced(b.ControllerStore.ExtractGVK(objtypes.ObjectTypeGateway))
	// Previous secrets referenced by a Gateway
	previousSecretsReferenced := b.ControllerStore.ReferencedObjects.PreviousReferencedSecrets.AllReferenced(b.ControllerStore.ExtractGVK(objtypes.ObjectTypeGateway))

	// Cert
	// Handle the newly added referenced Secrets
	// b.handleNewReferencedSecretsStorage(previousSecretsReferenced, currentSecretsReferenced)
	b.handleNewReferencedSecretsStorage(previousSecretsReferenced, currentSecretsReferenced)
	// Handle the Secrets that are de-referenced
	b.handleDeReferencedSecretsStorage(previousSecretsReferenced, currentSecretsReferenced)
	// Handle the Secrets that are still referenced, but migth have added : UPSERTED or DELETED
	b.handleUpdatedSecretsStorage(previousSecretsReferenced, currentSecretsReferenced)

	// crt-list
	b.handleCrtList(previousSecretsReferenced, currentSecretsReferenced)
}

func (b *CertificateBuilderImpl) ensureCertificatesStorage() {
	// ---------------
	// Write them on disk
	certUpdates := b.ControllerStore.CertificateUpdates
	crtListUpdates := b.ControllerStore.CrtListUpdates
	// ------------
	// cert
	for _, certData := range certUpdates.Created {
		_ = b.certStorage.WriteOnDisk(certData)
	}
	for _, certData := range certUpdates.Updated {
		_ = b.certStorage.WriteOnDisk(certData)
	}
	for _, certData := range certUpdates.Deleted {
		_ = b.certStorage.DeleteFromDisk(certData)
	}

	// -------------
	// crt-list
	for _, crtListData := range crtListUpdates.Created {
		_ = b.certStorage.WriteCrtListOnDisk(crtListData)
	}
	for _, crtListData := range crtListUpdates.Updated {
		_ = b.certStorage.WriteCrtListOnDisk(crtListData)
	}
	for _, crtListData := range crtListUpdates.Deleted {
		_ = b.certStorage.DeleteCrtListFromDisk(crtListData)
	}
}

func (*CertificateBuilderImpl) runtimeUpdates() {
}

// -----------
// cert

func (b *CertificateBuilderImpl) handleNewReferencedSecretsStorage(previousRefSecrets, newRefSecrets map[client.ObjectKey]map[client.ObjectKey]struct{}) {
	newlyReferenced := utils.SetDifference(newRefSecrets, previousRefSecrets)
	// -----------
	// Compute new CertificateData
	for nsName := range newlyReferenced {
		secret, ok := b.GateTree.Secrets[nsName]
		if !ok {
			b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "Cert [none-newref not found]", logging.LogAttrKey(nsName))
			continue
		}
		// In the same bach handling, the Gateway is updated with a new Secret, but the Secret is deleted
		// Do not create it
		if secret.TreeStatus.Status == store.StatusDeleted {
			b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "Cert [none-newref deleted]", logging.LogAttrKey(nsName))
			continue
		}

		certData, err := b.certStorage.NewCertificateData(secret.K8sResource)
		if err != nil {
			continue
		}
		b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "Cert [create-newref]", logging.LogAttrKey(nsName))
		b.ControllerStore.addCreatedCertificate(certData)
	}
}

func (b *CertificateBuilderImpl) handleDeReferencedSecretsStorage(previousRefSecrets, newRefSecrets map[client.ObjectKey]map[client.ObjectKey]struct{}) {
	deReferenced := utils.SetDifference(previousRefSecrets, newRefSecrets)
	for nsName := range deReferenced {
		certPath := b.certStorage.CertPath(nsName)
		certData := certificate.NewCertificateData(certPath, nil)
		b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "Cert [delete-unref]", logging.LogAttrKey(nsName))
		b.ControllerStore.addDeletedCertificate(certData)
	}
}

func (b *CertificateBuilderImpl) handleUpdatedSecretsStorage(previousRefSecrets, newRefSecrets map[client.ObjectKey]map[client.ObjectKey]struct{}) {
	secretsIntersection := utils.SetIntersection(previousRefSecrets, newRefSecrets)
	for secretKey := range secretsIntersection {
		// Is secret updated ?
		secretUpdate, ok := b.ClusterStore.Updates.Secrets[secretKey]
		if ok {
			switch secretUpdate.Status {
			case store.StatusUpserted:
				certData, err := b.certStorage.NewCertificateData(secretUpdate.NewObject)
				if err != nil {
					continue
				}
				// Add the impacted Gateways
				// Adding the impacted Frontends will be done in haproxycfg mgr
				if secretUpdate.OldObject == nil {
					// Secret CREATED
					b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "Cert [create-sameref]", logging.LogAttrKey(secretKey))
					b.ControllerStore.addCreatedCertificate(certData)
					continue
				}
				// Secret UPDATED
				b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "Cert [update-sameref]", logging.LogAttrKey(secretKey))
				b.ControllerStore.addUpdatedCertificate(certData)

			case store.StatusDeleted:
				b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "Cert [delete-sameref]", logging.LogAttrKey(secretKey))
				certPath := b.certStorage.CertPath(secretKey)
				certData := certificate.NewCertificateData(certPath, nil)
				b.ControllerStore.addDeletedCertificate(certData)
			default:
				b.Logger.LogAttrs(context.Background(), slog.LevelError, "handling secret [default]", logging.LogAttrKey(secretKey))
			}
		}
	}
}

// -----------
// crt-list

func (b *CertificateBuilderImpl) handleCrtList(previousRefSecrets, newRefSecrets map[client.ObjectKey]map[client.ObjectKey]struct{}) {
	previousSecretsByGatewayListener := secretsPerGatewayListener(previousRefSecrets)
	newSecretsByGatewayListener := secretsPerGatewayListener(newRefSecrets)

	b.handleNewReferencedCrtList(previousSecretsByGatewayListener, newSecretsByGatewayListener)
	b.handleDeReferencedCrtList(previousSecretsByGatewayListener, newSecretsByGatewayListener)
	b.handleUpdatedCrtList(previousSecretsByGatewayListener, newSecretsByGatewayListener)
}

// secretsPerGatewayListener returns for each gateway Key the list of secret Keys
func secretsPerGatewayListener(gatewaysPerSecret map[client.ObjectKey]map[client.ObjectKey]struct{}) map[client.ObjectKey]map[client.ObjectKey]struct{} {
	secretsPerGateway := make(map[client.ObjectKey]map[client.ObjectKey]struct{})
	for secretKey, gateways := range gatewaysPerSecret {
		for gatewayKey := range gateways {
			if _, ok := secretsPerGateway[gatewayKey]; !ok {
				secretsPerGateway[gatewayKey] = make(map[client.ObjectKey]struct{})
			}
			secretsPerGateway[gatewayKey][secretKey] = struct{}{}
		}
	}
	return secretsPerGateway
}

func (b *CertificateBuilderImpl) handleNewReferencedCrtList(previousSecretsPerGatewayListener, newSecretsPerGatewayListener map[client.ObjectKey]map[client.ObjectKey]struct{}) {
	newReferencedListeners := utils.SetDifference(newSecretsPerGatewayListener, previousSecretsPerGatewayListener)

	for listenerKey := range newReferencedListeners {
		// Secret Keys for this Gateway
		secretKeys := newSecretsPerGatewayListener[listenerKey]

		// Keep only existing secrets
		filteredSecretKeys := b.keepOnlyExistingSecrets(listenerKey, secretKeys)

		// Write the crt-list
		crtListData := b.certStorage.NewCrtListData(listenerKey, filteredSecretKeys)
		b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [create-newref]", logging.LogAttrKey(listenerKey))
		b.ControllerStore.addCreatedCrtList(crtListData)
	}
}

func (b *CertificateBuilderImpl) handleDeReferencedCrtList(previousSecretsByGatewayListener, newSecretsByGatewayListener map[client.ObjectKey]map[client.ObjectKey]struct{}) {
	deReferencedListeners := utils.SetDifference(previousSecretsByGatewayListener, newSecretsByGatewayListener)
	for listenerKey := range deReferencedListeners {
		crtListData := b.certStorage.NewCrtListData(listenerKey, nil)
		b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [delete-unref]", logging.LogAttrKey(listenerKey))
		b.addDeletedCrtList(crtListData)
	}
}

func (b *CertificateBuilderImpl) keepOnlyExistingSecrets(listenerKey client.ObjectKey, secretKeys map[client.ObjectKey]struct{}) map[client.ObjectKey]struct{} {
	res := make(map[client.ObjectKey]struct{})

	// Keep only existing secrets
	for secretKey := range secretKeys {
		if _, ok := b.GateTree.Secrets[secretKey]; !ok {
			b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [discard][non-existing]", logging.LogAttrKey(listenerKey),
				slog.String("secretKey", secretKey.String()))
			delete(secretKeys, secretKey)
			continue
		}
		if b.GateTree.Secrets[secretKey].TreeStatus.Status == store.StatusDeleted {
			b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [discard][deleted]", logging.LogAttrKey(listenerKey),
				slog.String("secretKey", secretKey.String()))
			delete(secretKeys, secretKey)
			continue
		}
		res[secretKey] = struct{}{}
		// b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [content][keep]", logging.LogAttrKey(listenerKey),
		// 	slog.String("secretKey", secretKey.String()))
	}
	return res
}

func (b *CertificateBuilderImpl) handleUpdatedCrtList(previousSecretsByGatewayListener, newSecretsByGatewayListener map[client.ObjectKey]map[client.ObjectKey]struct{}) {
	// Gateway listeners that were referencing secrets and still are...
	// But Secrets might have been created or deleted
	// Secret content change is ok, it's stored in Cert, not in crt-list
	listenersIntersection := utils.SetIntersection(previousSecretsByGatewayListener, newSecretsByGatewayListener)

	// Computing which one have an updated crt-list content (= list of secrets modified)
	for listenerKey := range listenersIntersection {
		previousSecretRefKeys := previousSecretsByGatewayListener[listenerKey]
		newSecretRefKeys := newSecretsByGatewayListener[listenerKey]

		// Added/Removed/Unchanged
		addedSecretRefsForListener := utils.SetDifference(newSecretRefKeys, previousSecretRefKeys)
		unchangedSecretRefsForListener := utils.SetIntersection(newSecretRefKeys, previousSecretRefKeys)

		// Build the list of crt-list
		crtlistUpdated := false

		// 1- Add the new cert if exisiting....
		secretKeys := make(map[client.ObjectKey]struct{})
		for secretKey := range addedSecretRefsForListener {
			b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [content][add-newref]", logging.LogAttrKey(listenerKey),
				slog.String("secretKey", secretKey.String()))
			secretKeys[secretKey] = struct{}{}
		}
		secretKeys = b.keepOnlyExistingSecrets(listenerKey, secretKeys)
		if len(secretKeys) != 0 {
			b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [updated][new entries]", logging.LogAttrKey(listenerKey))
			crtlistUpdated = true
		}

		// for secretKey := range unchangedSecretRefsForListener {
		// 	secretKeys[secretKey] = struct{}{}
		// }
		// 2- Checked the unchanged secretRefs.... they might have been created or deleted
		// Checks now the 2 lists of secrets: previousSecretKeys and updatedSecretKeys
		// If any difference, re-write the content
		// diff1 := utils.SetDifference(previousSecretRefKeys, unchangedSecretRefsForListener)
		// diff2 := utils.SetDifference(unchangedSecretRefsForListener, previousSecretRefKeys)
		// Remove non exists or deleted secrets from secrets to add

		// Also check if there are any newly created Secrets
		for secretKey := range unchangedSecretRefsForListener {
			treeSecret, ok := b.GateTree.Secrets[secretKey]
			if !ok {
				b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [content][skip-notfound]", logging.LogAttrKey(listenerKey),
					slog.String("secretKey", secretKey.String()))
				continue
			}
			if treeSecret.TreeStatus.Status == store.StatusUpserted {
				if treeSecret.TreeStatus.OldTreeResource == nil {
					b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [content][add-new]", logging.LogAttrKey(listenerKey),
						slog.String("secretKey", secretKey.String()))
					secretKeys[secretKey] = struct{}{}
					crtlistUpdated = true
					continue
				}
				b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [content][add-updated]", logging.LogAttrKey(listenerKey),
					slog.String("secretKey", secretKey.String()))
				crtlistUpdated = true
				secretKeys[secretKey] = struct{}{}
			}
			if treeSecret.TreeStatus.Status == store.StatusDeleted {
				b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [content][discard-deleted]", logging.LogAttrKey(listenerKey),
					slog.String("secretKey", secretKey.String()))
				crtlistUpdated = true
				continue
			}
			b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [content][add-unchanged]", logging.LogAttrKey(listenerKey),
				slog.String("secretKey", secretKey.String()))
			secretKeys[secretKey] = struct{}{}
		}

		// if len(diff1) != 0 || len(diff2) != 0 || crtlistUpdated {
		if crtlistUpdated {
			if len(secretKeys) != 0 {
				crtListData := b.certStorage.NewCrtListData(listenerKey, secretKeys)
				b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [update-content]", logging.LogAttrKey(listenerKey))
				b.ControllerStore.addUpdatedCrtList(crtListData)
			} else {
				crtListData := b.certStorage.NewCrtListData(listenerKey, nil)
				b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [delete-empty]", logging.LogAttrKey(listenerKey))
				b.ControllerStore.addDeletedCrtList(crtListData)
			}
		}
	}
}

// func (b *CertificateBuilderImpl) isListenerValid(listenerKey client.ObjectKey) bool {
// 	gatewayKey, listenerName, err := ConvertListenerKeyToGatewayKeyAndListenerName(listenerKey)
// 	if err != nil {
// 		return false
// 	}
// 	treeGw := b.GateTree.Gateways[gatewayKey]
// 	if treeGw == nil {
// 		return false
// 	}
// 	listener, ok := treeGw.Listeners[listenerName]
// 	if !ok {
// 		return false
// 	}
// 	return listener.Valid
// }
