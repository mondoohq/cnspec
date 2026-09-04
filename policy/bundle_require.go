// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"maps"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/providers"
	"go.mondoo.com/mql/utils/multierr"
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

// RequiredProviderNames returns the names of the providers the given
// requirements name, in order and without duplicates. We only pull
// requirements that are providers for now, expand when we add more types.
func RequiredProviderNames(require []*Requirement) []string {
	seen := make(map[string]struct{}, len(require))
	res := make([]string, 0, len(require))
	for _, req := range require {
		if req == nil || req.Provider == "" {
			continue
		}
		if _, ok := seen[req.Provider]; ok {
			continue
		}
		seen[req.Provider] = struct{}{}
		res = append(res, req.Provider)
	}
	return res
}

// RequiredProviders returns the names of every provider the bundle's policies
// and querypacks require.
func (p *Bundle) RequiredProviders() []string {
	var require []*Requirement
	for _, policy := range p.Policies {
		require = append(require, policy.Require...)
	}
	for _, pack := range p.Packs {
		require = append(require, pack.Require...)
	}
	return RequiredProviderNames(require)
}

// EnsureRequirements makes sure the resource schemas of all providers that
// the policies and querypacks in the bundle require are installed. Use it
// whenever the whole bundle is about to be compiled: compilation drops the
// queries it cannot resolve, so a schema that arrives later is too late for
// the checks that needed it.
//
// A scan against a bundle it did not compile itself does not need all of
// them, and pulling them is what broke scanning on platforms a provider ships
// no binary for. That path installs schemas per asset instead, through
// EnsureProviderSchemas (see policy/scan).
//
// Policies that do not specify any requirements are skipped (use the
// policy-missing-require lint rule to warn about those).
func (p *Bundle) EnsureRequirements(autoUpdate bool) error {
	return EnsureProviderSchemas(p.RequiredProviders(), autoUpdate)
}

// EnsureProviderSchemas installs the resource schemas of the named providers,
// and of the providers they depend on. It never downloads provider binaries:
// schemas are all that compiling a bundle and its asset filters needs, and
// unlike binaries they are published for every platform (some providers only
// ship binaries for a subset of platforms). The binaries a scan actually
// needs are installed per asset once its matching asset filters — and through
// them its applicable policies — are known (see policy/scan).
//
// Failures are collected rather than aborting on the first one: a provider
// that cannot be fetched only costs the queries that use its resources.
func EnsureProviderSchemas(names []string, autoUpdate bool) error {
	if len(names) == 0 {
		return nil
	}

	existing, err := providers.ListActive()
	if err != nil {
		return err
	}

	var errs multierr.Errors
	deps := map[string]providers.ProviderLookup{}
	for _, name := range names {
		providerDeps, err := ensureProviderSchema(name, autoUpdate, existing)
		if err != nil {
			if !autoUpdate {
				// only warn if auto update is disabled, as the user might want to manually install providers
				log.Warn().Err(err).Str("provider", name).Msg("failed to ensure provider schema")
				continue
			}
			errs.Add(err)
			continue
		}
		maps.Copy(deps, providerDeps)
	}

	// The dependencies are collected above and installed in one pass here,
	// once every provider named by the caller is in place. Providers share
	// dependencies (network alone is a dependency of aws, k8s, os, oci and
	// mondoo), and a dependency that is itself one of the named providers is
	// already installed by the loop above, so batching lets both cases fall
	// out as a single lookup instead of a download.
	errs.Add(installDependencySchemas(deps, autoUpdate, existing)...)

	return errs.Deduplicate()
}

// ensureProviderSchema makes sure the named provider's resource schema is
// available, installing it schema-only (without the binary) if it is missing.
// It returns the provider's own dependencies for the caller to install once
// all of them are known.
func ensureProviderSchema(name string, autoUpdate bool, existing providers.Providers) (map[string]providers.ProviderLookup, error) {
	provider := existing.Lookup(providers.ProviderLookup{ProviderName: name})
	if provider == nil {
		if !autoUpdate {
			return nil, errors.New("provider '" + name + "' is not installed")
		}
		nu, err := providers.InstallSchemaOnly(name, "")
		if err != nil {
			return nil, err
		}
		existing.Add(nu)
		provider = nu
	}
	return collectDependencySchemas(provider), nil
}

// collectDependencySchemas returns the providers the given provider depends
// on. Dependencies mostly matter at execution time (where the full install
// handles them), but their resources can appear in the bundle's queries and
// filters, so compilation needs their schemas too.
func collectDependencySchemas(provider *providers.Provider) map[string]providers.ProviderLookup {
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

	all := provider.Schema.AllDependencies()
	deps := make(map[string]providers.ProviderLookup, len(all))
	for _, dep := range all {
		deps[dep.Name] = providers.ProviderLookup{ID: dep.Id, ProviderName: dep.Name}
	}
	return deps
}

// installDependencySchemas installs the resource schemas of the collected
// dependencies that are still missing. One failure does not stop the rest:
// a dependency whose schema cannot be fetched only costs the queries that
// use its resources.
func installDependencySchemas(deps map[string]providers.ProviderLookup, autoUpdate bool, existing providers.Providers) []error {
	if !autoUpdate {
		return nil
	}

	var errs []error
	for _, dep := range deps {
		if existing.Lookup(dep) != nil {
			continue
		}
		nu, err := providers.InstallSchemaOnly(dep.ProviderName, "")
		if err != nil {
			errs = append(errs, multierr.Wrap(err, "failed to install dependency '"+dep.ProviderName+"'"))
			continue
		}
		existing.Add(nu)
	}
	return errs
}
