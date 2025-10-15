#!/bin/bash

set -e

SCRIPT_DIR=$(dirname "$0")

# Clean up previous runs
# docker stop k0s-hug-controller k0s-hug-worker &>/dev/null || true
# docker rm k0s-hug-controller k0s-hug-worker &>/dev/null || true
# docker network rm k0s-net &>/dev/null || true

# Create a network
docker network create --subnet=172.21.0.0/24 k0s-net

# Start the controller
docker run -d --name k0s-hug-controller --hostname k0s-hug-controller --privileged --ip 172.21.0.2 -v /var/lib/k0s -v /var/log/pods -v "$SCRIPT_DIR/k0s.yaml:/etc/k0s/k0s.yaml" -p 8443:6443 --net k0s-net docker.io/k0sproject/k0s:v1.34.1-k0s.0 k0s controller --single --config /etc/k0s/k0s.yaml

# Wait for the controller to be ready
echo -n "Waiting for controller to be ready "
until curl -s -k https://127.0.0.1:8443/version > /dev/null 2>&1; do
  echo -n "."
  sleep 2
done
echo " OK"

# Single-node cluster, no worker join token needed.

# Configure kubeconfig
echo "Configuring kubeconfig..."
docker exec k0s-hug-controller k0s kubeconfig admin > "$SCRIPT_DIR/k0s-kubeconfig.yaml"

# Use a temporary kubeconfig to extract credentials and save them to files
TMP_KUBECONFIG="$SCRIPT_DIR/k0s-kubeconfig.yaml"
CA_CERT_FILE="$SCRIPT_DIR/k0s-ca.crt"
CLIENT_CERT_FILE="$SCRIPT_DIR/k0s-client.crt"
CLIENT_KEY_FILE="$SCRIPT_DIR/k0s-client.key"

kubectl --kubeconfig="$TMP_KUBECONFIG" config view --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}' | base64 -d > "$CA_CERT_FILE"
kubectl --kubeconfig="$TMP_KUBECONFIG" config view --raw -o jsonpath='{.users[0].user.client-certificate-data}' | base64 -d > "$CLIENT_CERT_FILE"
kubectl --kubeconfig="$TMP_KUBECONFIG" config view --raw -o jsonpath='{.users[0].user.client-key-data}' | base64 -d > "$CLIENT_KEY_FILE"

# Add the new cluster, user, and context to the main kubeconfig file
KUBECONFIG_MAIN="$HOME/.kube/config"
touch "$KUBECONFIG_MAIN" # Ensure the file exists
chmod 600 "$KUBECONFIG_MAIN"
kubectl config --kubeconfig="$KUBECONFIG_MAIN" set-cluster k0s --server="https://127.0.0.1:8443" --certificate-authority="$CA_CERT_FILE" --embed-certs=true
kubectl config --kubeconfig="$KUBECONFIG_MAIN" set-credentials k0s-user --client-certificate="$CLIENT_CERT_FILE" --client-key="$CLIENT_KEY_FILE" --embed-certs=true
kubectl config --kubeconfig="$KUBECONFIG_MAIN" set-context k0s --cluster=k0s --user=k0s-user
kubectl config --kubeconfig="$KUBECONFIG_MAIN" use-context k0s

# Clean up all temporary files
rm -f "$TMP_KUBECONFIG" "$CA_CERT_FILE" "$CLIENT_CERT_FILE" "$CLIENT_KEY_FILE"

echo "Cluster is starting up. It might take a few minutes for the worker to be ready."
