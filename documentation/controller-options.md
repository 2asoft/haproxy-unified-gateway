# ![HAProxy](../assets/images/haproxy-weblogo-210x49.png "HAProxy")
## Kubernetes Controller

## Options

Multiple options can be combined

Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

// multiple options can be combined
controller, err := controller.New(opt.Option1(arg1), opt.Flag())
```

Available options:

| Function | Arguments |
| ---:|:--- |
| ControllerConf | `controllerConf`(types.NamespacedName) |
| ControllerName | `controllerName`(string) |
| ControllerPodConfig | `controllerPodConfig`(config.ControllerPodConfig) |
| GatewayClass | `gatewayClass`(*ast.ArrayType) |
| KubeConfig | `kubeconfig`(string) |
| LeaderElectionConfig | `leaderElection`(config.LeaderElectionConfig) |
| Logging | `level`(slog.Level), `allowedCategories`(*ast.ArrayType) |
| MetricsConfig | `metricsConfig`(config.MetricsConfig) |
| SyncPeriod | `syncPeriod`(time.Duration) |
| TreeChannel | `treeCh`(*ast.ChanType) |
| WhiteListNamespaces | `whitelistNs`(*ast.ArrayType) |

### ControllerConf


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.ControllerConf(controllerConf))
```

### ControllerName


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.ControllerName(controllerName))
```

### ControllerPodConfig


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.ControllerPodConfig(controllerPodConfig))
```

### GatewayClass


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.GatewayClass(gatewayClass))
```

### KubeConfig


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.KubeConfig(kubeconfig))
```

### LeaderElectionConfig


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.LeaderElectionConfig(leaderElection))
```

### Logging


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.Logging(level, allowedCategories))
```

### MetricsConfig

ControllerPodConfig sets the ControllerPodConfig of the controller.

Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.MetricsConfig(metricsConfig))
```

### SyncPeriod


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.SyncPeriod(syncPeriod))
```

### TreeChannel


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.TreeChannel(treeCh))
```

### WhiteListNamespaces


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.WhiteListNamespaces(whitelistNs))
```

