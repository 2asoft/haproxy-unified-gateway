# HAProxy Kubernetes Gateway Deployment

This example demonstrates how to deploy the HAProxy Kubernetes Gateway controller.

## What is being deployed

This example deploys the following resources:

* **Namespace** named `haproxy-unified-gateway` for all the controller-related resources.
* **ServiceAccount** named `haproxy-unified-gateway` for the controller Pod.
* **ClusterRole** and **ClusterRoleBinding** to grant the controller the necessary permissions to watch and manage resources in the cluster.
* **Deployment** named `haproxy-unified-gateway` that runs the gateway controller.
* **Service** named `haproxy-unified-gateway` of type `NodePort` to expose the gateway's HTTP, HTTPS, and stats ports.
* **HugConf** custom resource named `hugconf` to configure the logging levels for the gateway controller.

## How to deploy

```bash
kubectl apply -f .
```
