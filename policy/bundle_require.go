// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers"
	"go.mondoo.com/mql/v13/utils/multierr"
)

// HasRequirements returns true if any policy or querypack in the bundle has provider requirements defined.
func (p *Bundle) HasRequirements() bool {
	for _, policy := range p.Policies {
		if len(policy.Require) > 0 {
			return true
		}
	}
	for _, pack := range p.Packs {
		if len(pack.Require) > 0 {
			return true
		}
	}
	return false
}

// EnsureRequirements makes sure that all required providers for the policies
// in the bundle are installed. Policies that do not specify any requirements
// are skipped (use the policy-missing-require lint rule to warn about those).
func (p *Bundle) EnsureRequirements(autoUpdate bool) error {
	existing, err := providers.ListActive()
	if err != nil {
		return err
	}

	for _, policy := range p.Policies {
		for _, require := range policy.Require {
			// we only pull requirements that are providers for now, expand when we add more types
			if require.Provider == "" {
				continue
			}
			if _, err := providers.EnsureProvider(providers.ProviderLookup{ID: require.Id, ProviderName: require.Provider}, autoUpdate, existing); err != nil {
				if !autoUpdate {
					// only warn if auto update is disabled, as the user might want to manually install providers
					log.Warn().Str("provider", require.Provider).Msgf("failed to ensure policy requirements for policy %q", policy.Name)
				} else {
					return multierr.Wrap(err, "failed to validate policy '"+policy.Name+"'")
				}
			}
		}
	}

	for _, pack := range p.Packs {
		for _, require := range pack.Require {
			if require.Provider == "" {
				continue
			}
			if _, err := providers.EnsureProvider(providers.ProviderLookup{ID: require.Id, ProviderName: require.Provider}, autoUpdate, existing); err != nil {
				if !autoUpdate {
					log.Warn().Str("provider", require.Provider).Msgf("failed to ensure querypack requirements for querypack %q", pack.Name)
				} else {
					return multierr.Wrap(err, "failed to validate querypack '"+pack.Name+"'")
				}
			}
		}
	}

	return nil
}
