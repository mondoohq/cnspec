// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// The connection half of the stand-in provider: the constants the config names,
// and the connect-time reads that a fifth of the real tree does instead of
// reading anything in ParseCLI.
package connection

import (
	"os"
)

const (
	EnvToken       = "DEMO_TOKEN"
	DiscoveryAll   = "all"
	DiscoveryUnits = "units"

	OptionEndpoint = "endpoint"
	EnvEndpoint    = "DEMO_ENDPOINT"
	EnvTailnet     = "DEMO_TAILNET"
)

// getOptionValueFrom is the accessor shape zoom and tailscale use: the option
// name and the variable name side by side, one package away from ParseCLI.
func getOptionValueFrom(options map[string]string, envVar string, option string) (string, bool) {
	value := ""
	if v, ok := options[option]; ok && len(v) != 0 {
		value = v
	}
	if envVal := os.Getenv(envVar); envVal != "" {
		value = envVal
	}
	return value, len(value) != 0
}

// Endpoint is read at connect time rather than while parsing.
func Endpoint(options map[string]string) (string, bool) {
	return getOptionValueFrom(options, EnvEndpoint, OptionEndpoint)
}

// Tailnet reads a variable that backs an option no connector declares as a
// flag, which is a gap rather than an association.
func Tailnet(options map[string]string) (string, bool) {
	return getOptionValueFrom(options, EnvTailnet, "tailnet")
}
