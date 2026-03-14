# Metrics Authentication: None (Default)

This is the default mode. Metrics are served over plain HTTP with no authentication.

## When to Use

- Local development and testing
- Environments where the metrics port is not exposed outside the cluster
- Trusted network environments with network policies restricting access

## Setup

### 1. Controller Configuration

No additional flags are needed. The default `--metrics-auth=none` is applied automatically.

```yaml
# example/deploy/hug/controller.yaml (args section)
args:
  - --hugconf-crd=haproxy-unified-gateway/hugconf
  # metrics-auth defaults to "none", no need to specify
```

Or explicitly:

```yaml
args:
  - --hugconf-crd=haproxy-unified-gateway/hugconf
  - --metrics-auth=none
```

### 2. Deploy

Apply the standard deployment manifests:

```bash
kubectl apply -f example/deploy/hug/namespace.yaml
kubectl apply -f example/deploy/hug/rbac.yaml
kubectl apply -f example/deploy/hug/hugconf.yaml
kubectl apply -f example/deploy/hug/controller.yaml
```

### 3. Verify

Port-forward and check the metrics endpoint:

```bash
kubectl port-forward -n haproxy-unified-gateway svc/haproxy-unified-gateway 31060:31060
curl http://localhost:31060/metrics
```

You should see Prometheus metrics output including lines like:

```
# HELP hug_event_batch_total Total number of event batches processed.
# TYPE hug_event_batch_total counter
hug_event_batch_total 0
```

### 4. Prometheus Scrape Config

```yaml
scrape_configs:
  - job_name: 'hug'
    kubernetes_sd_configs:
      - role: endpoints
        namespaces:
          names:
            - haproxy-unified-gateway
    relabel_configs:
      - source_labels: [__meta_kubernetes_service_name, __meta_kubernetes_endpoint_port_name]
        action: keep
        regex: haproxy-unified-gateway;metrics
```

## Security Considerations

- The metrics endpoint is accessible to anyone who can reach the port
- Use Kubernetes NetworkPolicies to restrict access if needed:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-metrics-from-prometheus
  namespace: haproxy-unified-gateway
spec:
  podSelector:
    matchLabels:
      run: haproxy-unified-gateway
  policyTypes:
    - Ingress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: monitoring
      ports:
        - port: 31060
          protocol: TCP
```
