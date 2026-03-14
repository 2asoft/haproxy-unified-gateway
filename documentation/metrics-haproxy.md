# HAProxy Native Prometheus Metrics

HAProxy has a built-in Prometheus exporter that exposes detailed metrics about connections,
requests, backends, servers, and more. These metrics are separate from the controller metrics
(prefixed with `hug_`) and are served by HAProxy itself.

## How It Works

The default `haproxy.cfg` (located at `fs/usr/local/hug/haproxy.cfg`) includes a `stats` frontend
that serves both the HAProxy stats dashboard and Prometheus metrics:

```haproxy
frontend stats from haproxytech
  mode http
  http-request use-service prometheus-exporter if { path /metrics }
  stats enable
  stats uri /
  stats refresh 10s
  stats show-legends
```

- `GET /metrics` — Prometheus metrics (via `prometheus-exporter` service)
- `GET /` — HAProxy stats HTML dashboard

## Endpoint

The stats frontend binds to the **stats port**, which defaults to `31024` (configured via `--stats-port`).

| What | Port | Path |
|------|------|------|
| Prometheus metrics | 31024 | `/metrics` |
| Stats dashboard | 31024 | `/` |

Both example deployments expose this port as the `stat` container and service port.

## Verify

Port-forward and check:

```bash
kubectl port-forward -n haproxy-unified-gateway svc/haproxy-unified-gateway 31024:31024

# Prometheus metrics
curl http://localhost:31024/metrics

# Stats dashboard
open http://localhost:31024/
```

You should see standard HAProxy metrics like:

```
# HELP haproxy_process_current_connections Current number of connections on the process.
# TYPE haproxy_process_current_connections gauge
haproxy_process_current_connections 0
# HELP haproxy_frontend_current_sessions Current number of active sessions.
# TYPE haproxy_frontend_current_sessions gauge
haproxy_frontend_current_sessions{proxy="stats"} 1
```

## Key Metrics

HAProxy exposes hundreds of metrics. Here are some of the most useful ones:

### Process

| Metric | Description |
|--------|-------------|
| `haproxy_process_current_connections` | Current active connections |
| `haproxy_process_max_connections` | Maximum observed connections |
| `haproxy_process_nbthread` | Number of threads |
| `haproxy_process_uptime_seconds` | HAProxy uptime |

### Frontends

| Metric | Labels | Description |
|--------|--------|-------------|
| `haproxy_frontend_current_sessions` | `proxy` | Current active sessions |
| `haproxy_frontend_bytes_in_total` | `proxy` | Total bytes received |
| `haproxy_frontend_bytes_out_total` | `proxy` | Total bytes sent |
| `haproxy_frontend_http_requests_total` | `proxy` | Total HTTP requests |
| `haproxy_frontend_request_errors_total` | `proxy` | Total request errors |
| `haproxy_frontend_denied_connections_total` | `proxy` | Total denied connections |

### Backends

| Metric | Labels | Description |
|--------|--------|-------------|
| `haproxy_backend_current_sessions` | `proxy` | Current active sessions |
| `haproxy_backend_http_responses_total` | `proxy`, `code` | Total HTTP responses by status code |
| `haproxy_backend_connect_time_average` | `proxy` | Average connect time (ms) |
| `haproxy_backend_response_time_average` | `proxy` | Average response time (ms) |
| `haproxy_backend_active_servers` | `proxy` | Number of active servers |
| `haproxy_backend_status` | `proxy` | Backend status (UP/DOWN) |

### Servers

| Metric | Labels | Description |
|--------|--------|-------------|
| `haproxy_server_current_sessions` | `proxy`, `server` | Current active sessions |
| `haproxy_server_status` | `proxy`, `server` | Server status |
| `haproxy_server_weight` | `proxy`, `server` | Server weight |
| `haproxy_server_check_failures_total` | `proxy`, `server` | Total health check failures |

For the full list, see the [HAProxy Prometheus exporter documentation](https://www.haproxy.com/documentation/haproxy-configuration-tutorials/metrics/prometheus/).

## Filtering Metrics

HAProxy's prometheus-exporter supports server-side filtering via query parameters on the `/metrics` endpoint.
This is important in large deployments where the full metrics output can be very large (tens of thousands of lines).

### Scope Filtering

Use the `scope` query parameter to limit which metric categories are returned:

| Scope | Description |
|-------|-------------|
| `global` | Process-wide metrics |
| `frontend` | Frontend metrics |
| `listener` | Listener metrics (requires `option socket-stats` in the frontend) |
| `backend` | Backend metrics |
| `server` | Server metrics (highest cardinality) |
| `sticktable` | Stick table metrics |
| `*` | All scopes (default) |

Multiple scopes can be combined:

```bash
# Only global and frontend metrics
curl http://localhost:31024/metrics?scope=global&scope=frontend

# Everything except per-server metrics (reduces output significantly)
curl http://localhost:31024/metrics?scope=global&scope=frontend&scope=backend
```

In a Prometheus scrape config:

```yaml
scrape_configs:
  - job_name: 'haproxy'
    params:
      scope:
        - global
        - frontend
        - backend
    # ...
```

### Metric Name Filtering (HAProxy 3.0+)

Use the `metrics` query parameter to include or exclude specific metrics by name.
Prefix with `-` to exclude:

```bash
# Only fetch specific metrics
curl "http://localhost:31024/metrics?metrics=haproxy_frontend_status,haproxy_backend_status"

# Exclude a high-cardinality metric
curl "http://localhost:31024/metrics?metrics=-haproxy_server_check_status"
```

Include and exclude can be combined — only the resulting set is returned.

> **Note:** The `metrics` filter does not support regex — only exact metric names.

### Exclude Maintenance Servers

Use the `no-maint` parameter to exclude servers in maintenance mode. This is especially useful
with `server-template` where unused slots remain in maintenance and inflate the metrics output:

```bash
curl "http://localhost:31024/metrics?no-maint"
```

In a Prometheus scrape config:

```yaml
params:
  no-maint:
    - ""
```

### Extra Counters

Use `extra-counters` to export additional protocol-specific counters (HTTP/1, HTTP/2, HTTP/3, QUIC):

```bash
curl "http://localhost:31024/metrics?extra-counters"
```

### Combining Filters

All query parameters can be combined:

```bash
# Global + frontend + backend scopes, exclude maintenance servers, exclude a specific metric
curl "http://localhost:31024/metrics?scope=global&scope=frontend&scope=backend&no-maint&metrics=-haproxy_server_check_status"
```

### Prometheus-side Filtering

For filtering by proxy name (which the exporter does not support natively), use
`metric_relabel_configs` in your Prometheus scrape config:

```yaml
metric_relabel_configs:
  # Only keep metrics for specific backends
  - source_labels: [proxy]
    regex: 'my_backend_.*'
    action: keep
```

> **Note:** Prometheus-side relabeling happens after the scrape, so it reduces Prometheus storage
> but does not reduce the load on HAProxy itself. Prefer server-side `scope` and `metrics` filtering
> when possible.

## Prometheus Scrape Config

```yaml
scrape_configs:
  - job_name: 'haproxy'
    kubernetes_sd_configs:
      - role: endpoints
        namespaces:
          names:
            - haproxy-unified-gateway
    relabel_configs:
      - source_labels: [__meta_kubernetes_service_name, __meta_kubernetes_endpoint_port_name]
        action: keep
        regex: haproxy-unified-gateway;stat
```

If using the Prometheus Operator:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: haproxy-unified-gateway-haproxy
  namespace: haproxy-unified-gateway
spec:
  selector:
    matchLabels:
      run: haproxy-unified-gateway
  endpoints:
    - port: stat
      path: /metrics
```

## Two Scrape Targets

To get complete observability, configure Prometheus to scrape **both** endpoints:

| Job | Port | What you get |
|-----|------|-------------|
| `hug` | `31060` (metrics) | Controller metrics: batch processing, config generation, cert/map operations, reloads |
| `haproxy` | `31024` (stat) | HAProxy metrics: connections, request rates, backend health, latency, error codes |

## Configuring the Stats Port

The stats port can be changed via the `--stats-port` CLI flag (default: `1024`).
In the example deployments, the stats frontend is bound to port `31024` via the HAProxy configuration.

If you change it, update:
1. The bind port in your HAProxy configuration (Global/Defaults CRD or `haproxy.cfg`)
2. The `stat` containerPort and Service port in your deployment manifest
