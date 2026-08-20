// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/mqlc"
	"go.mondoo.com/mql/providers"
	"go.mondoo.com/mql/providers-sdk/v1/resources"
	"go.mondoo.com/mql/utils/multierr"
)

// Providers are installed lazily during a scan, in four steps:
//
//  1. The bundle installs the resource schemas its asset filters cannot be
//     compiled without, and no other requirement.
//  2. The providers serving the root assets' connections are fully installed
//     before discovery connects them.
//  3. Per asset, before the asset filters run, the providers whose resources
//     the filter queries reference are installed with binaries.
//  4. Once the matching filters are known, the providers required by the
//     policies that actually apply to the asset are installed.
//
// Steps 1 and 2 run in distributeJob, steps 3 and 4 per asset in runPolicy.
// Requirements of policies that do not apply are never installed; their
// providers may not even ship binaries for the platform cnspec runs on.

// providerInstallMu serializes provider installation across concurrently
// scanned assets. Installs write to the shared providers directory; two
// assets pulling the same provider at once would race.
var providerInstallMu sync.Mutex

// ensureProviderBinaries makes sure every given provider is fully installed,
// binary included. Providers whose schema was installed without a binary
// (see Bundle.EnsureRequirements) are completed by installing the full
// package for the version the schema was pulled at.
func ensureProviderBinaries(lookups []providers.ProviderLookup) error {
	if len(lookups) == 0 {
		return nil
	}

	providerInstallMu.Lock()
	defer providerInstallMu.Unlock()

	existing, err := providers.ListActive()
	if err != nil {
		return err
	}

	var errs multierr.Errors
	for _, lookup := range lookups {
		provider := existing.Lookup(lookup)
		if provider != nil && provider.Path == "" {
			// builtin providers have no binary to install
			continue
		}

		if provider != nil && !provider.HasBinary {
			// The schema was installed without the binary; complete the
			// installation at the same version.
			nu, err := providers.Install(provider.Name, provider.Version)
			if err != nil {
				errs.Add(multierr.Wrap(err, "failed to install provider '"+provider.Name+"'"))
				continue
			}
			existing.Add(nu)
		}

		// Installs the provider if it is missing entirely, and ensures the
		// dependencies of already installed providers.
		if _, err := providers.EnsureProvider(lookup, true, existing); err != nil {
			errs.Add(err)
		}
	}
	return errs.Deduplicate()
}

// filterProviderLookups compiles the given asset filter queries and collects
// the providers whose resources they reference. Filters that do not compile
// are skipped, mirroring executor.ExecuteFilterQueries.
func filterProviderLookups(schema resources.ResourcesSchema, filters []*policy.Mquery) []providers.ProviderLookup {
	conf := mqlc.NewConfig(schema, mql.DefaultFeatures)
	ids := map[string]struct{}{}
	for i := range filters {
		bundle, err := mqlc.Compile(filters[i].Mql, nil, conf)
		if err != nil || bundle.CodeV2 == nil {
			continue
		}
		for _, block := range bundle.CodeV2.Blocks {
			for _, chunk := range block.Chunks {
				if chunk.Call != llx.Chunk_FUNCTION || chunk.Id == "" {
					continue
				}
				if f := chunk.Function; f != nil && f.Binding != 0 {
					// a bound call, i.e. a field on an already created resource
					continue
				}
				info := schema.Lookup(chunk.Id)
				if info == nil || info.Provider == "" {
					continue
				}
				ids[info.Provider] = struct{}{}
			}
		}
	}

	res := make([]providers.ProviderLookup, 0, len(ids))
	for id := range ids {
		res = append(res, providers.ProviderLookup{ID: id})
	}
	return res
}

// ensureFilterProviders installs the providers needed to evaluate the given
// asset filters against this asset. Filters can only match if the providers
// serving their resources are able to run; which policies apply is not known
// yet at this point.
func (s *localAssetScanner) ensureFilterProviders(filters []*policy.Mquery) {
	if len(filters) == 0 || s.job.runtime == nil || !s.job.runtime.AutoUpdate.Enabled {
		return
	}
	lookups := filterProviderLookups(s.Runtime.Schema(), filters)
	if err := ensureProviderBinaries(lookups); err != nil {
		log.Warn().Err(err).Str("asset", s.job.Asset.Mrn).
			Msg("failed to install providers for asset filters; some policies may not be matched to this asset")
	}
}

// ensureApplicablePolicyProviders installs the providers required by the
// policies and querypacks that apply to this asset, i.e. those whose asset
// filters matched. Requirements of policies whose filters did not match are
// skipped, so a provider is not pulled just because its policy is enabled in
// the space — it may not even ship a binary for this platform.
//
// This reads the job's bundle, whose ComputedFilters are the ones computed
// upstream. That is the case the eager install broke. An incognito scan
// compiles its own copy of the bundle, so the job's copy has no computed
// filters there and every requirement conservatively counts as applicable —
// no worse than before, since a failed install is only a warning now.
func (s *localAssetScanner) ensureApplicablePolicyProviders(matched []*policy.Mquery) {
	if s.job.Bundle == nil || s.job.runtime == nil || !s.job.runtime.AutoUpdate.Enabled {
		return
	}

	matchedCodeIDs := make(map[string]struct{}, len(matched))
	matchedMql := make(map[string]struct{}, len(matched))
	for _, m := range matched {
		if m.CodeId != "" {
			matchedCodeIDs[m.CodeId] = struct{}{}
		}
		if m.Mql != "" {
			matchedMql[strings.TrimSpace(m.Mql)] = struct{}{}
		}
	}

	seen := map[string]struct{}{}
	var lookups []providers.ProviderLookup
	collect := func(require []*policy.Requirement) {
		for _, req := range require {
			if req.Provider == "" {
				continue
			}
			key := req.Id + "\x00" + req.Provider
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			lookups = append(lookups, providers.ProviderLookup{ID: req.Id, ProviderName: req.Provider})
		}
	}

	for _, pol := range s.job.Bundle.Policies {
		if len(pol.Require) == 0 {
			continue
		}
		if !filtersMatch(pol.ComputedFilters, matchedCodeIDs, matchedMql) {
			log.Debug().Str("policy", pol.Name).Str("asset", s.job.Asset.Mrn).
				Msg("skipping provider requirements, policy does not apply to this asset")
			continue
		}
		collect(pol.Require)
	}
	for _, pack := range s.job.Bundle.Packs {
		if len(pack.Require) == 0 {
			continue
		}
		if !filtersMatch(pack.ComputedFilters, matchedCodeIDs, matchedMql) {
			log.Debug().Str("querypack", pack.Name).Str("asset", s.job.Asset.Mrn).
				Msg("skipping provider requirements, querypack does not apply to this asset")
			continue
		}
		collect(pack.Require)
	}

	if err := ensureProviderBinaries(lookups); err != nil {
		log.Warn().Err(err).Str("asset", s.job.Asset.Mrn).
			Msg("failed to install providers required by applicable policies; affected checks will report errors")
	}
}

// filtersMatch reports whether any of the given computed filters is among the
// asset's matched filters. Filters are compared by code id (the Items map is
// keyed by it), falling back to the raw MQL. A policy without computed
// filters cannot be scoped to assets, so it conservatively counts as
// matching.
func filtersMatch(filters *policy.Filters, codeIDs, mqls map[string]struct{}) bool {
	if filters == nil || len(filters.Items) == 0 {
		return true
	}
	for key, item := range filters.Items {
		if _, ok := codeIDs[key]; ok {
			return true
		}
		if item == nil {
			continue
		}
		if item.CodeId != "" {
			if _, ok := codeIDs[item.CodeId]; ok {
				return true
			}
		}
		if item.Mql != "" {
			if _, ok := mqls[strings.TrimSpace(item.Mql)]; ok {
				return true
			}
		}
	}
	return false
}

// ensureFilterRequirements installs the resource schemas a scan needs in
// order to decide which policies apply to an asset — and nothing more.
//
// A requirement is what a policy's checks need in order to run, and whether
// they run at all is decided by the policy's asset filters. So a requirement
// is pulled here only when the policy's filters cannot be compiled without
// it: those filters have to be evaluated before anything else can be, and the
// policy's own requirements are the only providers we can offer them.
// Everything else waits for step 4 above.
//
// This is for a bundle the scan does not compile itself. A caller that goes
// on to compile the whole bundle needs Bundle.EnsureRequirements instead:
// compilation drops the queries it cannot resolve, so by the time the filters
// have run those checks are already gone.
func ensureFilterRequirements(bundle *policy.Bundle, autoUpdate bool) error {
	return policy.EnsureProviderSchemas(
		filterRequirements(bundle, providers.DefaultRuntime().Schema()),
		autoUpdate,
	)
}

// filterRequirements returns the providers required by the policies and
// querypacks whose asset filters cannot be compiled against the given schema.
func filterRequirements(bundle *policy.Bundle, schema resources.ResourcesSchema) []string {
	if bundle == nil {
		return nil
	}

	conf := mqlc.NewConfig(schema, mql.DefaultFeatures)
	var require []*policy.Requirement
	for _, pol := range bundle.Policies {
		if len(pol.Require) == 0 {
			continue
		}
		if assetFiltersCompile(conf, policyAssetFilters(pol)) {
			log.Debug().Str("policy", pol.Name).
				Msg("deferring provider requirements, the policy's asset filters can be evaluated without them")
			continue
		}
		require = append(require, pol.Require...)
	}
	for _, pack := range bundle.Packs {
		if len(pack.Require) == 0 {
			continue
		}
		if assetFiltersCompile(conf, packAssetFilters(pack)) {
			log.Debug().Str("querypack", pack.Name).
				Msg("deferring provider requirements, the querypack's asset filters can be evaluated without them")
			continue
		}
		require = append(require, pack.Require...)
	}
	return policy.RequiredProviderNames(require)
}

// policyAssetFilters returns the filters that decide whether the policy
// applies to an asset: the computed aggregate once the bundle has been
// compiled, and the authored group filters before that.
func policyAssetFilters(pol *policy.Policy) []*policy.Filters {
	res := make([]*policy.Filters, 0, len(pol.Groups)+1)
	res = append(res, pol.ComputedFilters)
	for _, group := range pol.Groups {
		if group != nil {
			res = append(res, group.Filters)
		}
	}
	return res
}

// packAssetFilters is policyAssetFilters for a querypack, which carries its
// authored filters on the pack itself as well as on its groups.
func packAssetFilters(pack *policy.QueryPack) []*policy.Filters {
	res := make([]*policy.Filters, 0, len(pack.Groups)+2)
	res = append(res, pack.ComputedFilters, pack.Filters)
	for _, group := range pack.Groups {
		if group != nil {
			res = append(res, group.Filters)
		}
	}
	return res
}

// assetFiltersCompile reports whether every given asset filter compiles
// against the providers that are installed right now. One that does not
// cannot be evaluated, and so cannot decide anything, until the provider
// holding its resources arrives.
//
// A policy with no asset filters at all compiles vacuously. That is the
// intended answer: it applies to every asset, so there is nothing to decide
// here and its requirements are pulled once the asset is known.
func assetFiltersCompile(conf mqlc.CompilerConfig, filters []*policy.Filters) bool {
	for _, set := range filters {
		if set == nil {
			continue
		}
		for _, item := range set.Items {
			if item == nil || item.Mql == "" {
				continue
			}
			if _, err := mqlc.Compile(item.Mql, nil, conf); err != nil {
				log.Debug().Err(err).Str("filter", item.Mql).
					Msg("asset filter does not compile with the installed providers")
				return false
			}
		}
	}
	return true
}
