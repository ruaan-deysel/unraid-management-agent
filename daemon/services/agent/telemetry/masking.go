package telemetry

import "regexp"

// sensitive matches values that must never reach span attributes.
var sensitive = regexp.MustCompile(`(?i)(registration key\s*\S+|wireguard\S*|password\s*[=:]\s*\S+|token\s*[=:]?\s*\S+|sk-[a-z0-9-]+|pk-lf-\S+|sk-lf-\S+)`)

// Mask redacts known-sensitive substrings from a string for safe tracing.
func Mask(s string) string {
	return sensitive.ReplaceAllString(s, "[REDACTED]")
}
