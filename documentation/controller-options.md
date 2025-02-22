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

| Function | Arguments |
| ---:|:--- |
| ControllerName | `controllerName`(string) |
| ControllerPodConfig | `controllerPodConfig`(config.ControllerPodConfig) |
| GatewayClass | `gatewayClass`(string) |
| LeaderElectionConfig | `leaderElection`(config.LeaderElectionConfig) |
| Logging | `level`(slog.Level) |
| MetricsConfig | `metricsConfig`(config.MetricsConfig) |
| RLogging |  |

### ControllerName


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/controller
  github.com/haproxytech/kubernetes-controller/controller/options
)

controller, err := controller.New(opt.ControllerName(controllerName))
```

### ControllerPodConfig


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/controller
  github.com/haproxytech/kubernetes-controller/controller/options
)

controller, err := controller.New(opt.ControllerPodConfig(controllerPodConfig))
```

### GatewayClass


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/controller
  github.com/haproxytech/kubernetes-controller/controller/options
)

controller, err := controller.New(opt.GatewayClass(gatewayClass))
```

### LeaderElectionConfig


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/controller
  github.com/haproxytech/kubernetes-controller/controller/options
)

controller, err := controller.New(opt.LeaderElectionConfig(leaderElection))
```

### Logging


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/controller
  github.com/haproxytech/kubernetes-controller/controller/options
)

controller, err := controller.New(opt.Logging(level))
```

### MetricsConfig

ControllerPodConfig sets the ControllerPodConfig of the controller.

Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/controller
  github.com/haproxytech/kubernetes-controller/controller/options
)

controller, err := controller.New(opt.MetricsConfig(metricsConfig))
```

### RLogging


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/controller
  github.com/haproxytech/kubernetes-controller/controller/options
)

controller, err := controller.New(opt.RLogging())
```

