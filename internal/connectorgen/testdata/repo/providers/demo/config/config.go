// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// A stand-in provider config, holding one instance of every shape the real tree
// uses: a constant from a sibling package, a discovery list built from
// constants, a flag whose Type and Option are enum identifiers, an Option
// spelled as an OR, and a Long assembled with fmt.Sprintf out of the very
// environment variable names the provider reads.
//
// It lives under testdata so the Go tool ignores it. It is never compiled, and
// it does not have to be: the extractor only parses.
package config

import (
	"fmt"

	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/demo/connection"
)

var Config = plugin.Provider{
	Name:            "demo",
	ID:              "go.mondoo.com/mql/providers/demo",
	ConnectionTypes: []string{"demo"},
	Connectors: []plugin.Connector{
		{
			Name:    "demo",
			Use:     "demo",
			Short:   "a demonstration target",
			Aliases: []string{"dem"},
			Long: fmt.Sprintf(`Use the demo provider.

Supply the token with %s.`, connection.EnvToken),
			MinArgs: 2,
			MaxArgs: 2,
			Discovery: []string{
				connection.DiscoveryAll,
				connection.DiscoveryUnits,
			},
			Flags: []plugin.Flag{
				{
					Long: "token",
					Type: plugin.FlagType_String,
					Desc: "API token",
				},
				{
					Long:   "password",
					Type:   plugin.FlagType_String,
					Option: plugin.FlagOption_Password | plugin.FlagOption_Required,
					Desc:   "account password",
				},
				{
					Long:        "region",
					Type:        plugin.FlagType_String,
					ConfigEntry: "-",
					Desc:        "region to query",
				},
				{
					Long: "endpoint",
					Type: plugin.FlagType_String,
					Desc: "API endpoint",
				},
				{
					Long: "organization",
					Type: plugin.FlagType_String,
					Desc: "organization",
				},
				{
					Long:   "legacy",
					Type:   plugin.FlagType_Bool,
					Option: plugin.FlagOption_Hidden,
					Desc:   "does nothing",
				},
			},
		},
		{
			Name:  "demolite",
			Use:   "demolite",
			Short: "the same target, without a token or an argument",
			Flags: []plugin.Flag{
				{
					Long: "endpoint",
					Type: plugin.FlagType_String,
					Desc: "API endpoint",
				},
			},
		},
	},
}
