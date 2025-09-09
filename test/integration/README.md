# Tools for degugging: envtest kubeconfig
While running integration tests, an envtest kubeconfig is generated in:
- `os.TempDir()`/kubeconfig


You can then inspect the objects with `kubectl` or `k9s`:
```
k9s --kubeconfig=/tmp/kubeconfig
```
