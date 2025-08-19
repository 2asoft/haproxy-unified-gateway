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
| BackendNameTemplate | `template`(string) |
| CacheReSyncPeriod | `syncPeriod`(time.Duration) |
| ControllerConfCRD | `controllerConf`(types.NamespacedName) |
| ControllerName | `controllerName`(string) |
| DefaultsSectionName | `name`(string) |
| DisableIPv4 |  |
| DisableIPv6 |  |
| FrontendNameTemplate | `template`(string) |
| HaproxyConfChannel | `treeCh`(*ast.ChanType) |
| HaproxyDirs | `dirs`(config.HaproxyDirs) |
| IPV4BindAddr | `addr`(string) |
| IPV6BindAddr | `addr`(string) |
| InitialStructured | `structured`(haproxy.Structured) |
| KubeConfig | `kubeconfig`(string) |
| LeaderElectionConfig | `leaderElectionEnabled`(bool) |
| LinkID | `template`(string) |
| Logging | `handlerType`(logging.LogHandlerType), `defaultLevel`(slog.Level), `logSettings`(*ast.MapType) |
| MetricsConfig | `metricsConfig`(config.MetricsConfig) |
| Namespaces | `namespaces`(*ast.ArrayType) |
| ServerNameTemplate | `template`(string) |
| StartupSyncPeriod | `syncPeriod`(time.Duration) |
| SyncPeriod | `syncPeriod`(time.Duration) |

### BackendNameTemplate


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.BackendNameTemplate(template))
```

### CacheReSyncPeriod


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.CacheReSyncPeriod(syncPeriod))
```

### ControllerConfCRD


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.ControllerConfCRD(controllerConf))
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

### DefaultsSectionName


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.DefaultsSectionName(name))
```

### DisableIPv4


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.DisableIPv4())
```

### DisableIPv6


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.DisableIPv6())
```

### FrontendNameTemplate


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.FrontendNameTemplate(template))
```

### HaproxyConfChannel


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.HaproxyConfChannel(treeCh))
```

### HaproxyDirs


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.HaproxyDirs(dirs))
```

### IPV4BindAddr


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.IPV4BindAddr(addr))
```

### IPV6BindAddr


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.IPV6BindAddr(addr))
```

### InitialStructured


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.InitialStructured(structured))
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

controller, err := controller.New(opt.LeaderElectionConfig(leaderElectionEnabled))
```

### LinkID


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.LinkID(template))
```

### Logging


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.Logging(handlerType, defaultLevel, logSettings))
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

### Namespaces


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.Namespaces(namespaces))
```

### ServerNameTemplate


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.ServerNameTemplate(template))
```

### StartupSyncPeriod


Example:
```go
import (
  github.com/haproxytech/kubernetes-controller/k8s/gate
  github.com/haproxytech/kubernetes-controller/k8s/gate/options
)

controller, err := controller.New(opt.StartupSyncPeriod(syncPeriod))
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

