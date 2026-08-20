// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"github.com/cockroachdb/errors"
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

// EnsureRequirements makes sure the resource schemas of all providers that
// the policies and querypacks in the bundle require are installed. It never
// downloads provider binaries: schemas are all that compiling the bundle and
// its asset filters needs, and unlike binaries they are published for every
// platform (some providers only ship binaries for a subset of platforms).
// The binaries a scan actually needs are installed per asset once its
// matching asset filters — and through them its applicable policies — are
// known (see policy/scan).
//
// Policies that do not specify any requirements are skipped (use the
// policy-missing-require lint rule to warn about those). Failures are
// collected per policy rather than aborting on the first one.
func (p *Bundle) EnsureRequirements(autoUpdate bool) error {
	existing, err := providers.ListActive()
	if err != nil {
		return err
	}

	var errs multierr.Errors
	for _, policy := range p.Policies {
		for _, require := range policy.Require {
			// we only pull requirements that are providers for now, expand when we add more types
			if require.Provider == "" {
				continue
			}
			if err := ensureRequirementSchema(require, autoUpdate, existing); err != nil {
				if !autoUpdate {
					// only warn if auto update is disabled, as the user might want to manually install providers
					log.Warn().Str("provider", require.Provider).Msgf("failed to ensure policy requirements for policy %q", policy.Name)
				} else {
					errs.Add(multierr.Wrap(err, "failed to validate policy '"+policy.Name+"'"))
				}
			}
		}
	}

	for _, pack := range p.Packs {
		for _, require := range pack.Require {
			if require.Provider == "" {
				continue
			}
			if err := ensureRequirementSchema(require, autoUpdate, existing); err != nil {
				if !autoUpdate {
					log.Warn().Str("provider", require.Provider).Msgf("failed to ensure querypack requirements for querypack %q", pack.Name)
				} else {
					errs.Add(multierr.Wrap(err, "failed to validate querypack '"+pack.Name+"'"))
				}
			}
		}
	}

	return errs.Deduplicate()
}

// ensureRequirementSchema makes sure the required provider's resource schema
// is available, installing it schema-only (without the binary) if it is
// missing, along with the schemas of the provider's dependencies.
func ensureRequirementSchema(require *Requirement, autoUpdate bool, existing providers.Providers) error {
	provider := existing.Lookup(providers.ProviderLookup{ID: require.Id, ProviderName: require.Provider})
	if provider == nil {
		if !autoUpdate {
			return errors.New("provider '" + require.Provider + "' is not installed")
		}
		nu, err := providers.InstallSchemaOnly(require.Provider, "")
		if err != nil {
			return err
		}
		existing.Add(nu)
		provider = nu
	}
	return ensureDependencySchemas(provider, autoUpdate, existing)
}

// ensureDependencySchemas installs the resource schemas of the providers the
// given provider depends on. Dependencies mostly matter at execution time
// (where the full install handles them), but their resources can appear in
// the bundle's queries and filters, so compilation needs their schemas too.
func ensureDependencySchemas(provider *providers.Provider, autoUpdate bool, existing providers.Providers) error {
	if provider.Path == "" {
		// builtin providers carry their schema in the binary
		return nil
	}
	if err := provider.LoadResources(); err != nil {
		log.Warn().Err(err).Str("provider", provider.Name).Msg("failed to load provider schema, unable to look up dependencies")
		return nil
	}
	if provider.Schema == nil {
		return nil
	}

	for _, dep := range provider.Schema.AllDependencies() {
		if existing.Lookup(providers.ProviderLookup{ID: dep.Id, ProviderName: dep.Name}) != nil {
			continue
		}
		if !autoUpdate {
			continue
		}
		nu, err := providers.InstallSchemaOnly(dep.Name, "")
		if err != nil {
			return err
		}
		existing.Add(nu)
	}
	return nil
}
