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
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/storage"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ Builder = &SecretBuilderImpl{}

type SecretBuilderImpl struct {
	ControllerStore
	certStorage storage.CertificateStorage
	// storeCertificatesOnDisk is a flag that indicates to the gate library to store certificates on disk
	storeCertificateOnDisk bool
}

func NewSecretBuilder(controllerStore ControllerStore, storeCertOnDisk bool, certStorage storage.CertificateStorage) Builder {
	return &SecretBuilderImpl{
		ControllerStore:        controllerStore,
		storeCertificateOnDisk: storeCertOnDisk,
		certStorage:            certStorage,
	}
}

// --------------------
// GateTree Updates
// --------------------

func (b *SecretBuilderImpl) ComputeTreeUpdates() {
	b.computeGateTreeUpdates()
	// b.computeCertificateDiffs()
	//
	//	if b.storeCertificateOnDisk {
	//		b.ensureCertsStorage()
	//		err := b.certStorage.DeleteEmptyCertsDir()
	//		if err != nil {
	//			b.Logger.LogAttrs(context.Background(), slog.LevelError, "error deleting empty cert dirs",
	//				logging.LogAttrError(err))
	//		}
	//	}
}

func (b *SecretBuilderImpl) computeGateTreeUpdates() {
	for secretKey, secretUpdate := range b.ClusterStore.Updates.Secrets {
		b.computeTreeSecretUpdate(secretKey, secretUpdate)
	}
}

func (b *SecretBuilderImpl) computeTreeSecretUpdate(secretKey client.ObjectKey, secretUpdate store.Update[*v1.Secret]) {
	treeSecret := b.GateTree.Secrets[secretKey]

	switch secretUpdate.Status {
	case store.StatusUpserted:
		if treeSecret != nil {
			treeSecret.SetAsUpserted(b.Logger, secretUpdate.NewObject)
		} else {
			treeSecret = NewSecret(secretUpdate.NewObject)
		}
		treeSecret.SetAsManaged(b.Logger, b.ControllerStore)
	case store.StatusDeleted:
		//	b.deleteCertificateFromDisk(treeSecret)
		if treeSecret != nil {
			treeSecret.SetAsDeleted(b.Logger)
		}

		// else nothing to do
		// It did not exists, it's deleted, noop
	}
}

// -----------------------------------------------

func (b *SecretBuilderImpl) CleanTreeUpdates() {
	for secretKey, treeSecret := range b.GateTree.Secrets {
		if treeSecret.TreeStatus.Status == store.StatusDeleted {
			delete(b.GateTree.Secrets, secretKey)
			continue
		}
		treeSecret.TreeStatus = TreeUpdate[Secret]{}
	}
	for secretKey, treeSecret := range b.UnmanagedGateTree.Secrets {
		if treeSecret.TreeStatus.Status == store.StatusDeleted {
			delete(b.GateTree.Secrets, secretKey)
			continue
		}
		treeSecret.TreeStatus = TreeUpdate[Secret]{}
	}
}

// // -----------------------------------------------

// func (b *SecretBuilderImpl) computeCertificateDiffs() {
// 	// Current secrets referenced by a Gateway
// 	currentSecretsReferenced := b.ControllerStore.ReferencedObjects.ReferencedSecrets.AllReferenced(b.ControllerStore.ExtractGVK(objtypes.ObjectTypeGateway))
// 	// Previous secrets referenced by a Gateway
// 	previousSecretsReferenced := b.ControllerStore.ReferencedObjects.PreviousReferencedSecrets.AllReferenced(b.ControllerStore.ExtractGVK(objtypes.ObjectTypeGateway))

// 	// Cert
// 	// Handle the newly added referenced Secrets
// 	b.handleNewReferencedSecretsStorage(previousSecretsReferenced, currentSecretsReferenced)
// 	// Handle the Secrets that are de-referenced
// 	b.handleDeReferencedSecretsStorage(previousSecretsReferenced, currentSecretsReferenced)
// 	// Handle the Secrets that are still referenced, but migth have added : UPSERTED or DELETED
// 	b.handleUpdatedSecretsStorage(previousSecretsReferenced, currentSecretsReferenced)

// 	// crt-list
// 	b.handleCrtList(previousSecretsReferenced, currentSecretsReferenced)
// }

// func (b *SecretBuilderImpl) ensureCertsStorage() {
// 	// ---------------
// 	// Write them on disk
// 	if b.storeCertificateOnDisk {

// 		// ------------
// 		// cert
// 		certUpdates := b.ControllerStore.CertificateUpdates

// 		for _, certData := range certUpdates.Created {
// 			_ = b.certStorage.WriteOnDisk(certData)
// 		}
// 		for _, certData := range certUpdates.Updated {
// 			_ = b.certStorage.WriteOnDisk(certData)
// 		}
// 		for _, certData := range certUpdates.Deleted {
// 			_ = b.certStorage.DeleteFromDisk(certData)
// 		}

// 		// -------------
// 		// crt-list
// 		crtListUpdates := b.ControllerStore.CrtListUpdates
// 		for _, crtListData := range crtListUpdates.Created {
// 			_ = b.certStorage.WriteCrtListOnDisk(crtListData)
// 		}
// 		for _, crtListData := range crtListUpdates.Updated {
// 			_ = b.certStorage.WriteCrtListOnDisk(crtListData)
// 		}
// 		for _, crtListData := range crtListUpdates.Deleted {
// 			_ = b.certStorage.DeleteCrtListFromDisk(crtListData)
// 		}
// 	}
// }

// // -----------
// // cert

// func (b *SecretBuilderImpl) handleNewReferencedSecretsStorage(previousRefSecrets, newRefSecrets map[client.ObjectKey]map[client.ObjectKey]struct{}) {
// 	newlyReferenced := utils.SetDifference(newRefSecrets, previousRefSecrets)
// 	// -----------
// 	// Compute new CertificateData
// 	for nsName := range newlyReferenced {
// 		secret, ok := b.GateTree.Secrets[nsName]
// 		if !ok {
// 			b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "Cert [none-newref not found]", logging.LogAttrKey(nsName))
// 			continue
// 		}
// 		// In the same bach handling, the Gateway is updated with a new Secret, but the Secret is deleted
// 		// Do not create it
// 		if secret.TreeStatus.Status == store.StatusDeleted {
// 			b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "Cert [none-newref deleted]", logging.LogAttrKey(nsName))
// 			continue
// 		}

// 		certData, err := b.certStorage.NewCertificateData(secret.K8sResource)
// 		if err != nil {
// 			continue
// 		}
// 		// Add the impacted Gateways
// 		// Adding the impacted Frontends will be done in haproxycfg mgr
// 		impactedGateways := newRefSecrets[nsName]
// 		certData.ImpactedGateways = impactedGateways
// 		b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "Cert [create-newref]", logging.LogAttrKey(nsName))
// 		b.ControllerStore.addCreatedCertificate(certData)
// 	}
// }

// func (b *SecretBuilderImpl) handleDeReferencedSecretsStorage(previousRefSecrets, newRefSecrets map[client.ObjectKey]map[client.ObjectKey]struct{}) {
// 	deReferenced := utils.SetDifference(previousRefSecrets, newRefSecrets)
// 	for nsName := range deReferenced {
// 		certPath := b.certStorage.CertPath(nsName)
// 		certData := certificate.NewCertificateData(certPath, nil)
// 		certData.ImpactedGateways = previousRefSecrets[nsName]
// 		b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "Cert [delete-unref]", logging.LogAttrKey(nsName))
// 		b.ControllerStore.addDeletedCertificate(certData)
// 	}
// }

// func (b *SecretBuilderImpl) handleUpdatedSecretsStorage(previousRefSecrets, newRefSecrets map[client.ObjectKey]map[client.ObjectKey]struct{}) {
// 	secretsIntersection := utils.SetIntersection(previousRefSecrets, newRefSecrets)
// 	for secretKey := range secretsIntersection {
// 		// Is secret updated ?
// 		secretUpdate, ok := b.ClusterStore.Updates.Secrets[secretKey]
// 		if ok {
// 			switch secretUpdate.Status {
// 			case store.StatusUpserted:
// 				certData, err := b.certStorage.NewCertificateData(secretUpdate.NewObject)
// 				if err != nil {
// 					continue
// 				}
// 				// Add the impacted Gateways
// 				// Adding the impacted Frontends will be done in haproxycfg mgr
// 				impactedGateways := newRefSecrets[secretKey]
// 				certData.ImpactedGateways = impactedGateways
// 				if secretUpdate.OldObject == nil {
// 					// Secret CREATED
// 					b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "Cert [create-sameref]", logging.LogAttrKey(secretKey))
// 					b.ControllerStore.addCreatedCertificate(certData)
// 					continue
// 				}
// 				// Secret UPDATED
// 				b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "Cert [update-sameref]", logging.LogAttrKey(secretKey))
// 				b.ControllerStore.addUpdatedCertificate(certData)

// 			case store.StatusDeleted:
// 				b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "Cert [delete-sameref]", logging.LogAttrKey(secretKey))
// 				certPath := b.certStorage.CertPath(secretKey)
// 				certData := certificate.NewCertificateData(certPath, nil)
// 				certData.ImpactedGateways = previousRefSecrets[secretKey]
// 				b.ControllerStore.addDeletedCertificate(certData)
// 			default:
// 				b.Logger.LogAttrs(context.Background(), slog.LevelError, "handling secret [default]", logging.LogAttrKey(secretKey))
// 			}
// 		}
// 	}
// }

// // -----------
// // crt-list

// func (b *SecretBuilderImpl) handleCrtList(previousRefSecrets, newRefSecrets map[client.ObjectKey]map[client.ObjectKey]struct{}) {
// 	previousSecretsByGatewayListener := secretsPerGatewayListener(previousRefSecrets)
// 	newSecretsByGatewayListener := secretsPerGatewayListener(newRefSecrets)

// 	b.handleNewReferencedCrtList(previousSecretsByGatewayListener, newSecretsByGatewayListener)
// 	b.handleDeReferencedCrtList(previousSecretsByGatewayListener, newSecretsByGatewayListener)
// 	b.handleUpdatedCrtList(previousSecretsByGatewayListener, newSecretsByGatewayListener)
// }

// // secretsPerGatewayListener returns for each gateway Key the list of secret Keys
// func secretsPerGatewayListener(gatewaysPerSecret map[client.ObjectKey]map[client.ObjectKey]struct{}) map[client.ObjectKey]map[client.ObjectKey]struct{} {
// 	secretsPerGateway := make(map[client.ObjectKey]map[client.ObjectKey]struct{})
// 	for secretKey, gateways := range gatewaysPerSecret {
// 		for gatewayKey := range gateways {
// 			if _, ok := secretsPerGateway[gatewayKey]; !ok {
// 				secretsPerGateway[gatewayKey] = make(map[client.ObjectKey]struct{})
// 			}
// 			secretsPerGateway[gatewayKey][secretKey] = struct{}{}
// 		}
// 	}
// 	return secretsPerGateway
// }

// func (b *SecretBuilderImpl) handleNewReferencedCrtList(previousSecretsPerGatewayListener, newSecretsPerGatewayListener map[client.ObjectKey]map[client.ObjectKey]struct{}) {
// 	newReferencedListeners := utils.SetDifference(newSecretsPerGatewayListener, previousSecretsPerGatewayListener)

// 	for listenerKey := range newReferencedListeners {
// 		// Secret Keys for this Gateway
// 		secretKeys := newSecretsPerGatewayListener[listenerKey]

// 		// Keep only existing secrets
// 		b.keepOnlyExistingSecrets(listenerKey, secretKeys)

// 		// Write the crt-list)
// 		crtListData := b.certStorage.NewCrtListData(listenerKey, secretKeys)
// 		b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [create-newref]", logging.LogAttrKey(listenerKey))
// 		b.ControllerStore.addCreatedCrtList(crtListData)
// 	}
// }

// func (b *SecretBuilderImpl) handleDeReferencedCrtList(previousSecretsByGatewayListener, newSecretsByGatewayListener map[client.ObjectKey]map[client.ObjectKey]struct{}) {
// 	deReferencedListeners := utils.SetDifference(previousSecretsByGatewayListener, newSecretsByGatewayListener)
// 	for listenerKey := range deReferencedListeners {
// 		crtListData := b.certStorage.NewCrtListData(listenerKey, nil)
// 		b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [delete-unref]", logging.LogAttrKey(listenerKey))
// 		b.addDeletedCrtList(crtListData)
// 	}
// }

// func (b *SecretBuilderImpl) keepOnlyExistingSecrets(listenerKey client.ObjectKey, secretKeys map[client.ObjectKey]struct{}) {
// 	// Keep only existing secrets
// 	for secretKey := range secretKeys {
// 		if _, ok := b.GateTree.Secrets[secretKey]; !ok {
// 			b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [content][remove-non-existing]", logging.LogAttrKey(listenerKey),
// 				slog.String("secretKey", secretKey.String()))
// 			delete(secretKeys, secretKey)
// 			continue
// 		}
// 		if b.GateTree.Secrets[secretKey].TreeStatus.Status == store.StatusDeleted {
// 			b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [content][remove-deleted]", logging.LogAttrKey(listenerKey),
// 				slog.String("secretKey", secretKey.String()))
// 			delete(secretKeys, secretKey)
// 			continue
// 		}
// 		// b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [content][keep]", logging.LogAttrKey(listenerKey),
// 		// 	slog.String("secretKey", secretKey.String()))
// 	}
// }

// func (b *SecretBuilderImpl) handleUpdatedCrtList(previousSecretsByGatewayListener, newSecretsByGatewayListener map[client.ObjectKey]map[client.ObjectKey]struct{}) {
// 	// Gateway listeners that were referencing secrets and still are...
// 	// But Secrets might have been created or deleted
// 	// Secret content change is ok, it's stored in Cert, not in crt-list
// 	listenersIntersection := utils.SetIntersection(previousSecretsByGatewayListener, newSecretsByGatewayListener)

// 	// Computing which one have an updated crt-list content (= list of secrets modified)
// 	for listenerKey := range listenersIntersection {
// 		previousSecretKeys := previousSecretsByGatewayListener[listenerKey]
// 		newSecretKeys := newSecretsByGatewayListener[listenerKey]

// 		// Added/Removed/Unchanged
// 		newSecretsForListener := utils.SetDifference(newSecretKeys, previousSecretKeys)
// 		unchangedSecretsForListener := utils.SetIntersection(newSecretKeys, previousSecretKeys)

// 		updatedSecretKeys := make(map[client.ObjectKey]struct{})
// 		for secretKey := range newSecretsForListener {
// 			updatedSecretKeys[secretKey] = struct{}{}
// 		}
// 		for secretKey := range unchangedSecretsForListener {
// 			updatedSecretKeys[secretKey] = struct{}{}
// 		}

// 		// Remove non exists or deleted secrets from secrets to add
// 		b.keepOnlyExistingSecrets(listenerKey, updatedSecretKeys)

// 		// Checks now the 2 lists of secrets: previousSecretKeys and updateSecretKeys
// 		// If any difference, re-write the content
// 		diff1 := utils.SetDifference(previousSecretKeys, updatedSecretKeys)
// 		diff2 := utils.SetDifference(updatedSecretKeys, previousSecretKeys)
// 		// Also check if there are any newly created Secrets
// 		newlyCreatedSecret := false
// 		for secretKey := range unchangedSecretsForListener {
// 			treeSecret, ok := b.GateTree.Secrets[secretKey]
// 			if !ok {
// 				// should not happen, already kep existing ones
// 				continue
// 			}
// 			if treeSecret.TreeStatus.Status == store.StatusUpserted && treeSecret.TreeStatus.OldTreeResource == nil {
// 				b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [content][secret-upserted]", logging.LogAttrKey(listenerKey),
// 					slog.String("secretKey", secretKey.String()))
// 				newlyCreatedSecret = true
// 			}
// 		}

// 		if len(diff1) != 0 || len(diff2) != 0 || newlyCreatedSecret {
// 			if len(updatedSecretKeys) != 0 {
// 				crtListData := b.certStorage.NewCrtListData(listenerKey, updatedSecretKeys)
// 				b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [update-sameref]", logging.LogAttrKey(listenerKey))
// 				b.ControllerStore.addUpdatedCrtList(crtListData)
// 			} else {
// 				crtListData := b.certStorage.NewCrtListData(listenerKey, nil)
// 				b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [delete-empty]", logging.LogAttrKey(listenerKey))
// 				b.ControllerStore.addDeletedCrtList(crtListData)
// 			}
// 		}
// 	}
// }

// func (b *SecretBuilderImpl) updateListsBasedOnExistingSecrets(listenerKey client.ObjectKey,
// 	newSecretsForListener, removedSecretsForListener, unchangedSecretForListener map[client.ObjectKey]struct{},
// ) {
// 	// Checks newly referenced secrets
// 	for secretKey := range newSecretsForListener {
// 		if _, ok := b.GateTree.Secrets[secretKey]; !ok {
// 			b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [content][remove-non-existing]", logging.LogAttrKey(listenerKey),
// 				slog.String("secretKey", secretKey.String()))
// 			delete(newSecretsForListener, secretKey)
// 		}
// 		if b.GateTree.Secrets[secretKey].TreeStatus.Status == store.StatusDeleted {
// 			b.Logger.LogAttrs(context.Background(), slog.LevelDebug, "crt-list [content][remove-deleted]", logging.LogAttrKey(listenerKey),
// 				slog.String("secretKey", secretKey.String()))
// 			delete(secretKeys, secretKey)
// 			continue
// 		}
// 	}
// 	// Checks unchanged references secrets
// 	for secretKey := range unchangedSecretForListener {
// 		secretUpdate, ok := b.ClusterStore.Updates.Secrets[secretKey]
// 		if ok {
// 			switch secretUpdate.Status {
// 			case store.StatusUpserted:
// 				newSecretsForListener[secretKey] = struct{}{}
// 			case store.StatusDeleted:
// 				removedSecretsForListener[secretKey] = struct{}{}
// 			default:
// 				b.Logger.LogAttrs(context.Background(), slog.LevelError, "handling secret [default]", logging.LogAttrKey(secretKey))
// 			}
// 		}
// 	}
// }
