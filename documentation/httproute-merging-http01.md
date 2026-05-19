# HTTPRoute cross-route path precedence and HTTP-01 challenge routing

## Status

Implemented on branch `aasoft/multi_route`.

This note documents the production problem, the implemented routing model, the Gateway API behavior it targets, operational validation, and remaining work.

## Problem

Gateway API allows multiple HTTPRoutes to attach to the same listener and hostname. For a request, HUG must consider matching rules across all applicable routes and choose the best match.

A common production case is:

- an application HTTPRoute on the HTTP listener with `PathPrefix /` that redirects HTTP traffic to HTTPS
- a cert-manager HTTP-01 solver HTTPRoute on the same HTTP listener and hostname with `Exact /.well-known/acme-challenge/<token>`

Expected behavior:

- `GET /` matches the application redirect
- `GET /.well-known/acme-challenge/<token>` matches the solver route and returns the solver response

Previous behavior:

- HUG selected one route for listener+hostname before path matching
- a catch-all redirect route could win before a more specific route was considered
- cert-manager propagation checks could receive `301`, `403`, `404`, or `503` instead of `200`
- logs could show `backend_not_found` for a valid solver route

## Gateway API behavior implemented

The relevant Gateway API rule is HTTPRoute match precedence across all rules on all applicable routes. Core path precedence is:

1. `Exact` path match
2. `PathPrefix` match with the largest number of characters
3. method match
4. largest number of header matches
5. largest number of query param matches
6. route creation timestamp and `{namespace}/{name}` tie-breakers

This implementation covers the explicit-host path portion of that behavior:

- exact path matches across different HTTPRoutes can beat same-host prefix/catch-all routes
- prefix path matches across different HTTPRoutes are looked up before the old single-route fallback
- prefix map entries are applied longest path first for both file-backed maps and runtime map updates

The implementation does not yet cover method/header/query precedence, route timestamp tie-breakers, wildcard/catch-all hostnames in the new planner path, or a product-level precedence policy for regular expressions.

## Routing model

The existing HTTP frontend pipeline was effectively:

1. request host -> selected listener
2. selected listener + host -> selected route
3. selected route + request path -> backend

That is efficient, but it collapses all same-host attached routes into one selected route before path precedence can be applied across routes.

The implemented pipeline adds listener+host+path maps before the existing fallback:

```text
listener_host_path_exact.map
listener_host_path_prefix.map
```

HTTP frontend rule order:

1. set request path and host variables
2. select listener
3. build `txn.listener_host_path = selected_listener_name + "/" + host + path`
4. look up `listener_host_path_exact.map` and set `txn.route` on exact match
5. look up `listener_host_path_prefix.map` with `ifnotexists`
6. continue through existing listener-route and route path maps as fallback
7. existing fallback maps use `ifnotexists`, so they do not overwrite a listener+host+path hit

This preserves the existing maps for wildcard hosts, regex paths, TLS passthrough, and compatibility fallback while adding the missing listener+host+path dimension for explicit-host exact and prefix paths.

## Planner and map data structures

HTTP map generation now goes through `httpRoutePlan` and `httpRouteCandidate`.

Each candidate carries:

- listener key
- legacy route value name
- route hostnames
- HTTPRoute match
- backend or redirect pseudo-backend value

The planner emits:

- legacy `path_exact.map`, `path_prefix.map`, and `path_regex.map` entries
- `listener_host_path_exact.map` entries for explicit-host `PathMatchExact`
- `listener_host_path_prefix.map` entries for explicit-host `PathMatchPathPrefix`

The same desired map state feeds disk writes and runtime map updates. Runtime map updates use the same deterministic key order as disk writes so longer prefixes are installed before shorter prefixes.

## Why this approach

The implementation keeps HUG's existing map-based request-time model and adds the missing routing dimension without replacing the whole frontend pipeline.

Benefits:

- exact ACME challenge paths can coexist with same-host catch-all redirects
- normal explicit-host prefix routes can be considered before single-route fallback
- existing wildcard-host, regex, TLS passthrough, redirect, and weighted-backend behavior remains on the existing fallback path
- map state still supports file-backed startup and runtime updates
- no cert-manager-specific logic is introduced

The invariant is not ACME-specific: path precedence must apply across route boundaries.

## cert-manager parentRef namespace note

In clusters where the Gateway and application live in different namespaces, cert-manager HTTP-01 solver routes must include the Gateway namespace in `parentRefs`:

```yaml
solvers:
  - http01:
      gatewayHTTPRoute:
        parentRefs:
          - name: shared-gateway
            namespace: hug-gateways
            kind: Gateway
```

This is separate from HUG route selection:

- without the namespace, the solver route does not attach to the Gateway
- without cross-route path precedence, a correctly attached solver route can still lose to a same-host redirect route

## Behavior tests

The implementation adds unit coverage for:

- generating listener+host+path planner entries from exact and prefix path matches
- excluding prefix routes from exact maps and exact routes from prefix maps
- checking frontend rule order: exact host-path map, prefix host-path map, then listener-route fallback
- sorting map entries with longer paths before shorter paths

Additional integration coverage should exercise actual HAProxy request behavior for:

1. same listener and host, `PathPrefix /` redirect plus exact HTTP-01 solver route
2. solver route created after the redirect route
3. solver route deletion after issuance, falling back to redirect
4. multiple exact paths in different HTTPRoutes
5. exact path precedence over prefix path across different HTTPRoutes
6. longer prefix path precedence over shorter prefix path across different HTTPRoutes
7. file-backed maps and runtime map updates

## Integration test execution notes

Integration tests require a real HAProxy binary. `task test` defaults to `/usr/local/sbin/haproxy`, which may not exist on all developer machines. Override it with `HAPROXY_BIN`:

```sh
HAPROXY_BIN=/usr/bin/haproxy task test PKG=./test/integration/httproute
```

The repository `justfile` provides convenience recipes that auto-detect `/usr/local/sbin/haproxy`, `/usr/sbin/haproxy`, or `/usr/bin/haproxy`:

```sh
just unit
just integration ./test/integration/httproute
just integration-container ./test/integration/httproute
```

The container recipe installs HAProxy inside a temporary Go container and sets `HAPROXY_BIN` to the container path. It is slower, but avoids host path differences.

## Operational validation

An earlier image with the exact-path behavior was deployed to one cluster using an isolated image tag and validated with real cert-manager Gateway HTTP-01 staging and production issuers.

Validated behavior:

- existing HTTP redirects continued to work
- cert-manager HTTP-01 staging canary issued while redirect route was present
- cert-manager HTTP-01 production canary issued while redirect route was present
- a production site using HTTP redirect plus HTTPS route issued successfully and served valid TLS

## Remaining work

A complete Gateway API route/rule merge implementation should extend the planner to cover:

- method/header/query match precedence
- route creation timestamp and `{namespace}/{name}` tie-breakers
- wildcard and catch-all hostname candidates in the planner path
- an explicit product policy for regular expression precedence
- conformance-style integration tests against actual HAProxy behavior
