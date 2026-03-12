// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utilsk8s

import (
	"strings"

	v3 "github.com/haproxytech/haproxy-unified-gateway/api/gate/v3"
	objtypes "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/object-types"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// IsFilterExtensionRefKindSupported checks if the provided filter ExtensionRef has a supported Group and Kind.
// It only supports `v3.Backend` resources.
func IsFilterExtensionRefKindSupported(extensionRef *gatewayv1.LocalObjectReference, extractGVK ExtractGVK) bool {
	backendGVK := extractGVK(objtypes.ObjectTypeBackend)

	if extensionRef == nil {
		return false
	}
	if !strings.EqualFold(backendGVK.Kind, string(extensionRef.Kind)) {
		return false
	}
	if backendGVK.Group != string(extensionRef.Group) {
		return false
	}
	return true
}

func IsFilterExtensionRefKindMergeType(extensionRef *gatewayv1.LocalObjectReference, extractGVK ExtractGVK) bool {
	backendGVK := extractGVK(objtypes.ObjectTypeBackend)
	if extensionRef == nil {
		return false
	}
	if !strings.EqualFold("MergeType", string(extensionRef.Kind)) {
		return false
	}
	if backendGVK.Group != string(extensionRef.Group) {
		return false
	}
	if !(strings.EqualFold("Override", string(extensionRef.Name)) || strings.EqualFold("Append", string(extensionRef.Name))) {
		return false
	}
	return true
}

// IsBackendRefGroupKindSupported checks if the provided HTTPRoute parent reference has a supported Group and Kind.
// It only supports `corev1.Service` resources.
func IsBackendRefGroupKindSupported(backendRef gatewayv1.BackendObjectReference, extractGVK ExtractGVK) bool {
	serviceGVK := extractGVK(objtypes.ObjectTypeService)

	if backendRef.Kind != nil && *backendRef.Kind != gatewayv1.Kind(serviceGVK.Kind) {
		return false
	}
	if backendRef.Group != nil && *backendRef.Group != gatewayv1.Group(serviceGVK.Group) {
		return false
	}
	return true
}

// IsParentRefGroupKindSupported checks if the provided HTTPRoute parent reference has a supported Group and Kind.
// It only supports `gatewayv1.Gateway` resources.
func IsParentRefGroupKindSupported(parentRef gatewayv1.ParentReference, extractGVK ExtractGVK) bool {
	gvk := extractGVK(objtypes.ObjectTypeGateway)

	if parentRef.Kind != nil && *parentRef.Kind != gatewayv1.Kind(gvk.Kind) {
		return false
	}
	if parentRef.Group != nil && *parentRef.Group != gatewayv1.Group(gvk.Group) {
		return false
	}
	return true
}

// IsSecretGroupKindSupported checks if the provided certificate reference has a supported Group and Kind.
// It only supports core `v1.Secret` resources.
func IsSecretGroupKindSupported(certRef gatewayv1.SecretObjectReference) bool {
	if certRef.Kind != nil && *certRef.Kind != "Secret" {
		return false
	}
	if certRef.Group != nil && *certRef.Group != "" {
		return false
	}
	return true
}

// IsGlobalRefGroupKindSupported checks if the provided Global reference has a supported Group and Kind.
// It only supports `v3.Global` resources.
func IsGlobalRefGroupKindSupported(globalRef v3.CRReference, extractGVK ExtractGVK) bool {
	globalGVK := extractGVK(objtypes.ObjectTypeGlobal)

	if globalRef.Kind != nil && *globalRef.Kind != gatewayv1.Kind(globalGVK.Kind) {
		return false
	}
	if globalRef.Group != nil && *globalRef.Group != gatewayv1.Group(globalGVK.Group) {
		return false
	}
	return true
}

// IsDefaultsRefGroupKindSupported checks if the provided Defaults reference has a supported Group and Kind.
// It only supports `v3.Defaults` resources.
func IsDefaultsRefGroupKindSupported(defaultsRef v3.CRReference, extractGVK ExtractGVK) bool {
	defaultsGVK := extractGVK(objtypes.ObjectTypeDefaults)

	if defaultsRef.Kind != nil && *defaultsRef.Kind != gatewayv1.Kind(defaultsGVK.Kind) {
		return false
	}
	if defaultsRef.Group != nil && *defaultsRef.Group != gatewayv1.Group(defaultsGVK.Group) {
		return false
	}
	return true
}
