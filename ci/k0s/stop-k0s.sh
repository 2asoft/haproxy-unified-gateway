#!/bin/bash

#set -e

SCRIPT_DIR=$(dirname "$0")

echo "Stopping and removing k0s container..."
docker rm -f k0s-hug-controller &>/dev/null || true
sleep 2
echo "Removing k0s network..."
docker network rm k0s-net &>/dev/null || true

echo "Cleaning up kubeconfig..."
# Unset current context if it's k0s, then delete the context, user and cluster
if [ "$(kubectl config current-context 2>/dev/null)" == "k0s" ]; then
  kubectl config unset current-context &>/dev/null || true
fi
kubectl config delete-context k0s &>/dev/null || true
kubectl config delete-user k0s-user &>/dev/null || true
kubectl config delete-cluster k0s &>/dev/null || true
rm -f "$SCRIPT_DIR/k0s-kubeconfig.yaml" "$SCRIPT_DIR/k0s-ca.crt" "$SCRIPT_DIR/k0s-client.crt" "$SCRIPT_DIR/k0s-client.key"

echo "Cleanup complete."
