## Gateway API

HAProxy Unified Gateway currently supports v1.3.0 version, fot time being only specific routes are implemented:

- gateway.networking.k8s.io/v1
  - kind: HTTPRoute

## How to deploy

```sh
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml
```
