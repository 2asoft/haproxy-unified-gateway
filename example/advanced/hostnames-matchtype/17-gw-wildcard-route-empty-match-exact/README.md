# Echo Deployment

This example contains an example with:
- Gateway Listeners:
  - Wildcard: `*.haproxy`
- Route:
  - Empty
- MatchType:
  - Exact: `/api/`

## How to deploy

```sh
kubectl apply -f .
```

## How to test e2e curls from your computer

### Success


```sh
curl --header "Host: offload.haproxy" http://127.0.0.1:31081/api/
curl --header "Host: offload.haproxy"  https://127.0.0.1:31444/api/ -k
curl --header "Host: other.haproxy" http://127.0.0.1:31081/api/
```

### Failure

```sh
curl --header "Host: offload.haproxy" http://127.0.0.1:31081/api/foo
curl --header "Host: offload.haproxy"  https://127.0.0.1:31444/api/foo -k
curl --header "Host: other.haproxy" http://127.0.0.1:31081/api/foo

```
