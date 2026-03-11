# Global CustomResource

You can customize the `global` section of the generated `haproxy.cfg` by applying a `Global` Custom Resource.

The CRD schema can be found here: [Global CRD](../../api/definition/gate.v3.haproxy.org_globals.yaml)

---

## Basic example

```yaml
apiVersion: gate.v3.haproxy.org/v3
kind: Global
metadata:
  name: global
  namespace: example
spec:
  merge_strategy: append
  performance_options:
    maxconn: 32100
```

The `spec` accepts all fields (some are excluded see below) from the HAProxy `global` section (via the embedded
`models.Global` from client-native), plus the required `merge_strategy` field
described below.

The following fields are intentionally excluded and cannot be set
(they are managed by the controller or not applicable in Kubernetes):

- `daemon`
- `localpeer`
- `master-worker`
- `pidfile`
- `stats_timeout`
- `default_path`
- `tune_lua_options.bool_sample_conversion`
- `lua_options.load_per_thread`

Those fields are removed from the Global CRD and then can be not set.


---

## Pointing to a Global CR from HugConf

Reference the `Global` CR from your `HugConf` using the `globalRef` field:

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
  globalRef:
    group: gate.v3.haproxy.org
    kind: Global
    name: global
    namespace: example
```

| `globalRef` field | Description |
|---|---|
| `name` | Name of the `Global` CR (required) |
| `namespace` | Namespace of the `Global` CR. Defaults to the `HugConf` namespace when omitted. |
| `group` | API group — `gate.v3.haproxy.org` |
| `kind` | Resource kind — `Global` |

Removing `globalRef` from `HugConf` (or deleting the referenced CR) restores the
HUG built-in defaults defined in [`default_global.yaml`](../../hug/haproxy/default_cr/default_global.yaml).

- group
- kind
- namespace

are optional.

If namespace is omitted, it will take the same namespace where the `HugConf` is deployed.

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
  globalRef:
    name: global
```


---

## Merge pipeline

When a `Global` CR is applied, settings are resolved in three stages:

```
HUG defaults  ──► merge with Global CR (merge_strategy)  ──► apply mandatory settings
```

1. **HUG defaults** — a built-in baseline defined in
   [`default_global.yaml`](../../hug/haproxy/default_cr/default_global.yaml)
   that includes SSL ciphers, Lua options, log targets, a default `maxconn`, etc.
   These same defaults are also restored when the `Global` CR is deleted or
   `globalRef` is removed from `HugConf`.
2. **Global CR** — merged over the defaults according to `merge_strategy`.
3. **Mandatory settings** — always enforced on top of the result (see below).

---

## merge_strategy

The `merge_strategy` field controls how the CR is merged over the HUG defaults.
It must be one of `override` or `append`. The difference only matters for **list
fields** (e.g. `runtime_apis`, `log_target_list`); scalar fields always follow
the same rule: a non-zero value in the CR replaces the default.

### `override`

List fields in the CR **replace** the corresponding default list entirely.

```yaml
apiVersion: gate.v3.haproxy.org/v3
kind: Global
metadata:
  name: global
  namespace: example
spec:
  merge_strategy: override
  performance_options:
    maxconn: 32100
  log_target_list:
    - address: /dev/log
      facility: local0
      format: rfc5424
```

HUG default `log_target_list` (stdout/daemon) is **replaced** by the single
`/dev/log` entry above. `maxconn` is raised from `32000` to `32100`.

Result (before mandatory pass):

```
global
    maxconn 32100
    log /dev/log format rfc5424 local0
    ...
```

### `append`

List fields in the CR are **appended** to the default list.

```yaml
apiVersion: gate.v3.haproxy.org/v3
kind: Global
metadata:
  name: global
  namespace: example
spec:
  merge_strategy: append
  performance_options:
    maxconn: 32100
  log_target_list:
    - address: /dev/log
      facility: local0
      format: rfc5424
```

The `/dev/log` entry is added **after** the existing stdout entry; both are kept.
`maxconn` is raised just as with `override`.

Result (before mandatory pass):

```
global
    maxconn 32100
    log stdout    daemon    raw
    log /dev/log  local0    rfc5424
    ...
```

---

## Mandatory settings

After the CR is merged, HUG unconditionally enforces a set of mandatory fields
defined in [`mandatory_global.yaml`](../../hug/haproxy/mandatory/mandatory_global.yaml):

```yaml
hard_stop_after: 1800000
pidfile: <hug-managed path>
runtime_apis:
  - address: <hug-managed socket>
    expose_fd_listeners: true
    level: admin
```

The enforcement rules differ between scalars and lists:

### Scalar fields

Mandatory scalar values **always win** — they override whatever the CR or the
defaults produced. `hard_stop_after` and `pidfile` cannot be changed via a
`Global` CR.

### `runtime_apis` list

The mandatory runtime API socket (used internally by HUG to communicate with
HAProxy) is always placed **at position 0** in the final list. Entries from the
CR whose address differs from the mandatory one are preserved and appended after
it. Any entry with the same address as the mandatory one is deduplicated.

Example: if the CR adds a second socket:

```yaml
spec:
  merge_strategy: append
  runtime_apis:
    - address: /var/run/my-extra.sock
      level: user
```

Final `runtime_apis` order:

```
1. /var/run/haproxy/haproxy.sock   (mandatory, always first)
2. /var/run/my-extra.sock          (from CR)
```

> **Note:** Mandatory settings cannot be overridden by the `Global` CR regardless
> of the chosen `merge_strategy`.
