# ![HAProxy](../assets/images/haproxy-weblogo-210x49.png "HAProxy")
## Kubernetes Controller

## Options

Multiple options can be combined

Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/controller
  github.com/haproxytech/kubernetes-controller/controller/options
)

// multiple options can be combined
controller, err := controller.New(opt.Option1(arg1), opt.Flag())
```

Available options:

| Option | Arguments |
| ---:|:--- |
| GatewayClass | `string`(gatewayClass) |

### GatewayClass

Host sets the host of the controller.

The host can be an IP address or a hostname.

Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/controller
  github.com/haproxytech/kubernetes-controller/controller/options
)

controller, err := controller.New(opt.GatewayClass(gatewayClass))
```

