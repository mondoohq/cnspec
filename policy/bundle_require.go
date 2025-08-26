// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/providers"
	"go.mondoo.com/cnquery/v12/utils/multierr"
)

func (p *Bundle) EnsureRequirements(installIfNoRequire bool) error {
	existing, err := providers.ListActive()
	if err != nil {
		return err
	}

	var missingRequires []string
	for _, policy := range p.Policies {
		if len(policy.Require) == 0 && installIfNoRequire {
			missingRequires = append(missingRequires, policy.Name)
			continue
		}

		for _, require := range policy.Require {
			if err := ensureRequirement(require, existing); err != nil {
				return multierr.Wrap(err, "failed to validate policy '"+policy.Name+"'")
			}
		}
	}

	if len(missingRequires) != 0 {
		log.Info().Strs("policies", missingRequires).Msg("policy doesn't specify required providers, defaulting to installing all default providers")
		for _, v := range providers.DefaultProviders {
			if _, err := providers.EnsureProvider(providers.ProviderLookup{ID: v.ID}, true, nil); err != nil {
				return err
			}
		}
	}

	return nil
}

func ensureRequirement(require *Requirement, existing providers.Providers) error {
	res := existing.Lookup(providers.ProviderLookup{
		ID:           require.Id,
		ProviderName: require.Name,
	})
	if res != nil {
		return nil
	}

	if require.Id != "" {
		if !strings.HasPrefix(require.Id, "go.mondoo.com/cnquery/") {
			return errors.New("cannot install providers by ID that are not in the Mondoo releases at this time")
		}

		idx := strings.LastIndex(require.Id, "/")
		require.Name = require.Id[idx+1:]
	}

	if require.Name != "" {
		installed, err := providers.Install(require.Name, "")
		if err != nil {
			return multierr.Wrap(err, "failed to install "+require.Name)
		}
		providers.PrintInstallResults([]*providers.Provider{installed})
		return nil
	}

	return errors.New("found an empty `require` statement")
}
