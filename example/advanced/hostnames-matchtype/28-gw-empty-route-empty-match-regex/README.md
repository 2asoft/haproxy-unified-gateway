# Echo Deployment

This example contains an example with:
- Gateway Listeners:
  - Empty
- Route:
  - Empty
- MatchType:
  - Regex: `^/api/.*`

## How to deploy

```sh
kubectl apply -f .
```

## How to test e2e curls from your computer

### Success


### Success


```sh
curl --header "Host: offload.haproxy" http://127.0.0.1:31081/api/foo
curl --header "Host: offload.haproxy"  https://127.0.0.1:31444/api/foo -k
curl --header "Host: other.haproxy" http://127.0.0.1:31081/api/foo
curl --header "Host: other.haproxy"  https://127.0.0.1:31444/api/foo -k

```
