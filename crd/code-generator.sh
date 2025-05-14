#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# copy root go.mod and replace module name
echo "Copying root go.mod"
cp ../go.mod go.mod

### Adjusting module name
MODULE_NAME="github.com/haproxytech/kubernetes-controller"
echo "Adjusting module name"
sed -i 's|^module ${MODULE_NAME}.*|module ${MODULE_NAME}/crd|' go.mod
echo "Tidying modules"
go mod tidy

exit

# set client native version
# client_native_version=$(cat go.mod | grep "haproxy-controller/client-native-ee/v6" | awk '{print $NF}')
# #client_native_version="1.2.3.4"
# echo "Client Native Version: $client_native_version"
# for file in api-ee/ingress/v3/*.go; do
#     echo "$file"
#     # Use sed to replace the version string in Go files with the new version
#     sed -i  "s@// +kubebuilder:metadata:annotations=\"haproxy.com/client-native=.*\"@// +kubebuilder:metadata:annotations=\"haproxy.com/client-native=$client_native_version\"@" $file
# done

# code-generator build native, versioned clients, informers and other helpers
# via Kubernetes code generators from k8s.oi/code-generator

GOBIN="$(go env GOBIN)"
gopath="$(go env GOPATH)"
gobin="${GOBIN:-$(go env GOPATH)/bin}"

CR_DIR=$(dirname "$0")
OUTPUT_DIR="${CR_DIR}/.generated"
HDR_FILE="${CR_DIR}/../assets/license-header.txt"
CR_PKG="github.com/haproxytech/kubernetes-ingress/crs"
API_PKGS=$(find ${CR_DIR}/api-ee -mindepth 2 -type d -printf "$CR_PKG/api-ee/%P\n"| sort | tr '\n' ',')
API_PKGS=${API_PKGS::-1} # remove trailing ","

# Install Kubernetes Code Generators from k8s.io/code-generator

VERSION=$(go list -m  k8s.io/api  | cut -d ' ' -f2)

# new version is completly broken (with breaking changes \o/) use old one
#go install k8s.io/code-generator/cmd/{deepcopy-gen,register-gen,client-gen,lister-gen,informer-gen,defaulter-gen}@$VERSION
go install k8s.io/code-generator/cmd/{deepcopy-gen,register-gen,client-gen,lister-gen,informer-gen,defaulter-gen}@v0.29.5

# Generate Code
IFS=','
for API_PKG in $API_PKGS; do

    echo "Generating code for $API_PKG"
    GOPATH=$gopath "${gobin}/deepcopy-gen"\
    -O zz_generated.deepcopy\
        --input-dirs "${API_PKG}"\
        --go-header-file ${HDR_FILE}\
        --output-base ${OUTPUT_DIR}

    echo "Generating register funcs"
    GOPATH=$gopath "${gobin}/register-gen"\
    -O zz_generated.register\
        --input-dirs "${API_PKG}"\
        --go-header-file ${HDR_FILE}\
        --output-base ${OUTPUT_DIR}

    CR_VERSION=${API_PKG#"$CR_PKG/"}
    echo "Generating clientset"
    GOPATH=$gopath "${gobin}/client-gen"\
        --plural-exceptions "Defaults:Defaults"\
        --clientset-name "versioned"\
        --input "${API_PKG}"\
        --input-base "" \
        --output-package "${CR_PKG}/generated-ee/${CR_VERSION}/clientset"\
        --go-header-file ${HDR_FILE}\
        --output-base ${OUTPUT_DIR}\

    echo "Generating listers"
    GOPATH=$gopath "${gobin}/lister-gen"\
        --plural-exceptions "Defaults:Defaults"\
        --input-dirs "${API_PKGS}"\
        --output-package "${CR_PKG}/generated-ee/${CR_VERSION}/listers"\
        --go-header-file ${HDR_FILE}\
        --output-base ${OUTPUT_DIR}

    echo "Generating informers"
        GOPATH=$gopath "${gobin}/informer-gen"\
            --plural-exceptions "Defaults:Defaults"\
            --input-dirs "${API_PKG}"\
            --versioned-clientset-package "${CR_PKG}/generated-ee/${CR_VERSION}/clientset/versioned"\
            --listers-package "${CR_PKG}/generated-ee/${CR_VERSION}/listers"\
            --output-package "${CR_PKG}/generated-ee/${CR_VERSION}/informers"\
            --go-header-file ${HDR_FILE}\
            --output-base ${OUTPUT_DIR}
done


# Copy generated code into the right location
# This extra step is required because code generator seems to be working only in a GOPATH environment.
# https://github.com/kubernetes/code-generator/issues/57
# So the work around is to generate to OUTPUT_DIR then move code to the right location
for API_PKG in $API_PKGS; do
    CR_VERSION=${API_PKG#"$CR_PKG/"}
    mkdir -p ${CR_DIR}/generated-ee/${CR_VERSION}
    cp -r ${OUTPUT_DIR}/${CR_PKG}/generated-ee/${CR_VERSION}/{clientset,listers,informers} ${CR_DIR}/generated-ee/${CR_VERSION}
done
rm -rf ${OUTPUT_DIR}

CONTROLLER_GEN_VERSION=$(go list -m  sigs.k8s.io/controller-tools  | cut -d ' ' -f2)
go install sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_GEN_VERSION}

# # Controller-gen version
echo "Controller-gen: " ${CONTROLLER_GEN_VERSION}
controller-gen crd paths=./api-ee/ingress/v3/...  output:crd:dir=./definition-ee
# remove code-gen annotation (dependabot fails)
find ./definition-ee -type f -name '*.yaml' -exec sed -i '/controller-gen.kubebuilder.io\/version/d' {} +


# cleanup go.mod and go.sum
echo "Cleaning up go.mod and go.sum"
rm go.mod
rm go.sum


# # Removal of some fields from the CRDs
# # For example, for now we remove servers from the backend CRD v3
echo "Removing fields from the generated CRDs"
sh remove-fields-from-crds.sh
