package definition

import _ "embed"

//go:embed gate.v3.haproxy.org_backends.yaml
var backends []byte

//go:embed gate.v3.haproxy.org_hugconfs.yaml
var hugConf []byte

//go:embed gate.v3.haproxy.org_huggates.yaml
var hugGates []byte

//go:embed gate.v3.haproxy.org_globals.yaml
var globals []byte

// getCRDs returns a map of CRD definitions
func getCRDs() map[string][]byte {
	return map[string][]byte{
		"backends.gate.v3.haproxy.org": backends,
		"global.gate.v3.haproxy.org":   globals,
		"huggates.gate.v3.haproxy.org": hugGates,
		"hugconfs.gate.v3.haproxy.org": hugConf,
	}
}
