// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package pack

import (
	_ "embed"

	"go.mondoo.com/cnspec/v12/policy"
)

// sbomQueryPack is the embedded SBOM query pack YAML
//
//go:embed sbom.mql.yaml
var sbomQueryPack []byte

// QueryPack returns the SBOM query pack as a policy bundle.
// The bundle contains queries for collecting software inventory
// information from an asset including OS packages, Python packages,
// npm packages, and kernel information.
func QueryPack() (*policy.Bundle, error) {
	bundle, err := policy.BundleFromYAML(sbomQueryPack)
	if err != nil {
		return nil, err
	}
	// Convert the query pack to a policy bundle for execution
	bundle.ConvertQuerypacks()
	return bundle, nil
}
