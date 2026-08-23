// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// The parsing half of the stand-in provider. Every function below is one shape
// the extractor has to get right, and the comments say which.
package provider

import (
	"os"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/demo/connection"
)

type Service struct {
	*plugin.Service
}

// stringFlag is the variadic accessor: one flag, then a fallback list of
// variables it tries in order.
func stringFlag(flags map[string]*llx.Primitive, name string, envs ...string) string {
	if x, ok := flags[name]; ok && len(x.Value) != 0 {
		return string(x.Value)
	}
	for _, e := range envs {
		if v := os.Getenv(e); v != "" {
			return v
		}
	}
	return ""
}

// flagValue is the accessor that names a flag and nothing else. Its caller
// supplies the variable.
func flagValue(flags map[string]*llx.Primitive, name string) string {
	if x, ok := flags[name]; ok && len(x.Value) != 0 {
		return string(x.Value)
	}
	return ""
}

// envPassword returns a variable and takes no argument that says so, which is
// the shape mistral uses.
func envPassword() string {
	return strings.TrimSpace(os.Getenv("DEMO_PASSWORD"))
}

func (s *Service) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	flags := req.Flags

	conf := &inventory.Config{
		Type:    req.Connector,
		Options: map[string]string{},
	}

	// The longhand chain, with the reused `x` that makes scope matter: the
	// second binding must not give --token the organization's variable.
	token := ""
	if x, ok := flags["token"]; ok && len(x.Value) != 0 {
		token = string(x.Value)
	}
	if token == "" {
		token = os.Getenv(connection.EnvToken)
	}

	organization := ""
	if x, ok := flags["organization"]; ok && len(x.Value) != 0 {
		organization = string(x.Value)
	}
	if organization == "" {
		// Two variables joined rather than chosen between, which is okta's
		// shape: setting only the first one produces a broken value.
		organization = strings.TrimSpace(os.Getenv("DEMO_ORG_NAME")) + "." + strings.TrimSpace(os.Getenv("DEMO_BASE_URL"))
	}

	// The variadic accessor: a fallback list, not a composition.
	region := stringFlag(flags, "region", "DEMO_REGION", "DEMO_REGION_ID")

	// The half accessor plus an inline fallback.
	password := flagValue(flags, "password")
	if password == "" {
		password = envPassword()
	}

	// A value assembled out of two others. Nothing here may make --region look
	// as though it travelled in DEMO_PASSWORD.
	client := newClient(region, password)
	_ = client

	// The sub-command vocabulary, which exists nowhere but here.
	switch req.Args[0] {
	case "unit":
		conf.Options["unit"] = req.Args[1]
	case "group":
		conf.Options["group"] = req.Args[1]
	default:
		return nil, errNoSuchKind
	}

	conf.Options["token"] = token
	conf.Options["organization"] = organization
	conf.Options["region"] = region

	asset := inventory.Asset{Connections: []*inventory.Config{conf}}
	return &plugin.ParseCLIRes{Asset: &asset}, nil
}
