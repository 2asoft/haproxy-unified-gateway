# Echo Deployment

This example demonstrates how to deploy an echo server.

## What is being deployed

This example deploys the following resources:

* **Deployment** named `http-echo` that runs the echo server.
* **Service** named `http-echo` to expose the echo server's HTTP and HTTPS ports.

## How to deploy

```sh
kubectl apply -f .
```
