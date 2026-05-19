# Integration test notes

## HAProxy binary path

Integration tests require a real HAProxy binary. The Taskfile default is `/usr/local/sbin/haproxy`, but some systems install HAProxy at `/usr/bin/haproxy` or `/usr/sbin/haproxy`.

Override the path explicitly when needed:

```sh
HAPROXY_BIN=/usr/bin/haproxy task test PKG=./test/integration/httproute
```

The repository `justfile` also provides helper recipes:

```sh
just integration ./test/integration/httproute
just integration-container ./test/integration/httproute
```

`just integration` auto-detects common HAProxy locations unless `HAPROXY_BIN` is set. `just integration-container` runs the tests in a disposable Go container and installs HAProxy inside the container.

# Tools for debugging: envtest kubeconfig

While running integration tests, an envtest kubeconfig is generated in:
- `cfgDir/test folder/kubeconfig`

The exact location can be found at the beginning of the test logs:
```
=== RUN   TestGatewayTLSMultipleTestSuite
    /home/helene/go/src/gitlab.int.haproxy.com/haproxy-unified-gateway/test/integration/base/hugconfig.go:37: Haproxy config path: /tmp/hug/e2e-tests-gateway-tls-multiple
    /home/helene/go/src/gitlab.int.haproxy.com/haproxy-unified-gateway/test/integration/base/inttest.go:146: kubeconfig path: /tmp/hug/e2e-tests-gateway-tls-multiple/kubeconfig
```

You can then inspect the objects with `kubectl` or `k9s`:
```
k9s --kubeconfig=/tmp/hug/e2e-tests-gateway-tls-multiple
```

# Parallel run

The test in each folder under `test/integration` will be run in parallel in a different envtest.
Inside each folder, they run by default sequentially.

Let's take an example for `test/integration/gateway-tls-multiple`

Output of this tests will be in:
```
/tmp/hug/e2e-tests-gateway-tls-multiple
```

```
$ ls
ls
certlists  certs  haproxy.cfg  haproxy-master.sock  haproxy.pid  haproxy-runtime-api.sock  kubeconfig  maps  route.lua

```

Note also the kubeconfig file there.
