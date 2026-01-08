// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/providers"
	"go.mondoo.com/cnquery/v12/utils/multierr"
)

// HasRequirements returns true if any policy in the bundle has provider requirements defined.
func (p *Bundle) HasRequirements() bool {
	for _, policy := range p.Policies {
		if len(policy.Require) > 0 {
			return true
		}
	}
	return false
}

// EnsureRequirements makes sure that all required providers for the policies
// in the bundle are installed. If `installIfNoRequire` is true, it will install
// default providers for policies that do not specify any requirements.
func (p *Bundle) EnsureRequirements(installIfNoRequire bool, autoUpdate bool) error {
	existing, err := providers.ListActive()
	if err != nil {
		return err
	}

	var missingRequires []string
	for _, policy := range p.Policies {

		// collect policies that do not specify any requirements and install default providers later
		if len(policy.Require) == 0 && installIfNoRequire {
			missingRequires = append(missingRequires, policy.Name)
			continue
		}

		for _, require := range policy.Require {
			// we only pull requirements that are providers for now, expand when we add more types
			if require.Provider == "" {
				continue
			}
			if _, err := providers.EnsureProvider(providers.ProviderLookup{ID: require.Id, ProviderName: require.Provider}, autoUpdate, existing); err != nil {
				return multierr.Wrap(err, "failed to validate policy '"+policy.Name+"'")
			}
		}
	}

	// install default providers for policies that do not specify any requirements
	if len(missingRequires) != 0 {
		log.Debug().Strs("policies", missingRequires).Msg("policy doesn't specify required providers, defaulting to installing all default providers")
		for _, v := range providers.DefaultProviders {
			if _, err := providers.EnsureProvider(providers.ProviderLookup{ID: v.ID}, autoUpdate, nil); err != nil {
				return err
			}
		}
	}

	return nil
}
