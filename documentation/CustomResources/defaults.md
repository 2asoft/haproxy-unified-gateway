# Defaults CustomResource

You can customize the `defaults haproxytech` section of the generated `haproxy.cfg` by applying a `Defaults` Custom Resource.

The CRD schema can be found here: [Defaults CRD](../../api/definition/gate.v3.haproxy.org_defaults.yaml)

---

## Naming requirement

> **Important:** Only the `Defaults` CR whose **`spec.name`** equals **`haproxytech`** is managed by HUG.
> Any `Defaults` CR with a different `spec.name` is silently ignored and a warning is logged.

This constraint exists because HUG maps the `Defaults` CR directly to the HAProxy `defaults` section
named `haproxytech` in the generated configuration. There can only be one active `Defaults` CR at a time.

The Kubernetes `metadata.name` of the CR can be anything; it is the `spec.name` field that
identifies which HAProxy defaults section the CR targets.

---

## Basic example

```yaml
apiVersion: gate.v3.haproxy.org/v3
kind: Defaults
metadata:
  name: my-defaults        # K8s metadata name — can be anything
  namespace: example
spec:
  merge_strategy: append
  name: haproxytech        # HAProxy defaults section name — must be "haproxytech"
  client_timeout: 60000
  server_timeout: 60000
```

The `spec` accepts all fields from the HAProxy `defaults` section (via the embedded
`models.Defaults` from client-native), plus the required `merge_strategy` field
described below.

---

## Pointing to a Defaults CR from HugConf

Reference the `Defaults` CR from your `HugConf` using the `defaultsRef` field:

```yaml
apiVersion: gate.v3.haproxy.org/v3
kind: HugConf
metadata:
  name: hugconf
  namespace: haproxy-unified-gateway
spec:
  logging:
    defaultLevel: "Debug"
    categoryLevelList:
      - category: "k8s"
        level: "Info"
  defaultsRef:
    group: gate.v3.haproxy.org
    kind: Defaults
    name: my-defaults      # K8s metadata.name of the Defaults CR
    namespace: example
```

| `defaultsRef` field | Description |
|---|---|
| `name` | K8s `metadata.name` of the `Defaults` CR (required) |
| `namespace` | Namespace of the `Defaults` CR. Defaults to the `HugConf` namespace when omitted. |
| `group` | API group — `gate.v3.haproxy.org` |
| `kind` | Resource kind — `Defaults` |

Removing `defaultsRef` from `HugConf` (or deleting the referenced CR) restores the
HUG built-in defaults defined in [`default_defaults.yaml`](../../hug/haproxy/default_cr/default_defaults.yaml).

`group`, `kind`, and `namespace` are optional.

If `namespace` is omitted, it defaults to the namespace where `HugConf` is deployed.

So the shortest version is:
```yaml
apiVersion: gate.v3.haproxy.org/v3
kind: HugConf
metadata:
  name: hugconf
  namespace: haproxy-unified-gateway
spec:
  logging:
    defaultLevel: "Debug"
    categoryLevelList:
      - category: "k8s"
        level: "Info"
  defaultsRef:
    name: my-defaults      # K8s metadata.name of the Defaults CR
```

---

## Merge pipeline

When a `Defaults` CR is applied, settings are resolved in two stages:

```
HUG defaults  ──► merge with Defaults CR (merge_strategy)
```

1. **HUG defaults** — a built-in baseline defined in
   [`default_defaults.yaml`](../../hug/haproxy/default_cr/default_defaults.yaml).
   These same defaults are also restored when the `Defaults` CR is deleted or
   `defaultsRef` is removed from `HugConf`.
2. **Defaults CR** — merged over the built-in defaults according to `merge_strategy`.

Unlike the `Global` CR, there are no mandatory settings enforced after the merge.

---

## merge_strategy

The `merge_strategy` field controls how the CR is merged over the HUG defaults.
It must be one of `override` or `append`. The difference only matters for **list
fields** (e.g. `log_target_list`); scalar fields always follow the same rule:
a non-zero value in the CR replaces the default.

### `override`

List fields in the CR **replace** the corresponding default list entirely.

```yaml
apiVersion: gate.v3.haproxy.org/v3
kind: Defaults
metadata:
  name: my-defaults
  namespace: example
spec:
  merge_strategy: override
  name: haproxytech
  client_timeout: 60000
  log_target_list:
    - global: false
      address: /dev/log
      facility: local0
```

HUG default `log_target_list` (`global: true`) is **replaced** by the single
`/dev/log` entry above. `client_timeout` is raised from `50000` to `60000`.

Result:

```
defaults haproxytech
    client_timeout  60000
    log /dev/log local0
    ...
```

### `append`

List fields in the CR are **appended** to the default list.

```yaml
apiVersion: gate.v3.haproxy.org/v3
kind: Defaults
metadata:
  name: my-defaults
  namespace: example
spec:
  merge_strategy: append
  name: haproxytech
  client_timeout: 60000
  log_target_list:
    - global: false
      address: /dev/log
      facility: local0
```

The `/dev/log` entry is added **after** the existing `global: true` entry; both are kept.
`client_timeout` is raised just as with `override`.

Result:

```
defaults haproxytech
    client_timeout  60000
    log global
    log /dev/log local0
    ...
```

---

## HUG built-in defaults

The built-in baseline applied before the CR merge is:

```yaml
client_timeout: 50000
connect_timeout: 5000
dontlognull: enabled
http_keep_alive_timeout: 60000
http_request_timeout: 5000
log_format: '%ci:%cp [%tr] %ft %b/%s %TR/%Tw/%Tc/%Tr/%Ta %ST %B %CC %CS %tsc %ac/%fc/%bc/%sc/%rc %sq/%bq %hr %hs "%HM %[var(txn.base)] %HV"'
log_target_list:
  - global: true
name: haproxytech
queue_timeout: 5000
server_timeout: 50000
tunnel_timeout: 3600000
```
