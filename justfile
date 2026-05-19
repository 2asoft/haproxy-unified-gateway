set shell := ["bash", "-uc"]

haproxy_bin := env_var_or_default("HAPROXY_BIN", `if command -v haproxy >/dev/null 2>&1; then command -v haproxy; elif [ -x /usr/local/sbin/haproxy ]; then echo /usr/local/sbin/haproxy; elif [ -x /usr/sbin/haproxy ]; then echo /usr/sbin/haproxy; elif [ -x /usr/bin/haproxy ]; then echo /usr/bin/haproxy; else echo /usr/local/sbin/haproxy; fi`)

default:
    just --list

# Run non-integration Go tests with the environment expected by k8s/gate/options tests.
unit:
    POD_IP=127.0.0.1 POD_NAMESPACE=hug POD_NAME=hug-test \
      go test $(go list ./... | grep -v '/test/integration') -count=1

# Run integration tests against a host HAProxy binary. Override with HAPROXY_BIN=/path/to/haproxy.
integration pkg="./test/integration/...":
    HAPROXY_BIN={{haproxy_bin}} \
      POD_IP=127.0.0.1 POD_NAMESPACE=hug POD_NAME=hug-test ENVTEST_VERSION=v1.35.0 \
      go test {{pkg}} -count=1 -v

# Run integration tests in a disposable container with HAProxy installed in the container.
integration-container pkg="./test/integration/...":
    docker run --rm \
      -v "$PWD":/src \
      -v "$HOME/go/pkg/mod":/go/pkg/mod \
      -w /src \
      golang:1.26-alpine \
      sh -lc 'apk add --no-cache haproxy git build-base && HAPROXY_BIN=/usr/sbin/haproxy POD_IP=127.0.0.1 POD_NAMESPACE=hug POD_NAME=hug-test ENVTEST_VERSION=v1.35.0 go test {{pkg}} -count=1 -v'
