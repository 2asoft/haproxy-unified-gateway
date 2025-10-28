package utils //revive:disable:var-naming

import "strings"

// IsDomainWildcard checks if a domain is a wildcard domain (starts with "*.")
func IsDomainWildcard(domain string) bool {
	return strings.HasPrefix(domain, "*.")
}
