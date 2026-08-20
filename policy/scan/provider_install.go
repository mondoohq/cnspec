// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/v13/policy"
	"go.mondoo.com/mql/v13"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/mqlc"
	"go.mondoo.com/mql/v13/providers"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
	"go.mondoo.com/mql/v13/utils/multierr"
)

// Providers are installed lazily during a scan, in two steps per asset:
// first the ones needed to evaluate the asset filters, then — once the
// matching filters are known — the ones required by the policies that
// actually apply to the asset. Bundle.EnsureRequirements only installs
// resource schemas, so this is where binaries are pulled. Requirements of
// policies that do not apply are never installed; their providers may not
// even ship binaries for the platform cnspec runs on.

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

// connectionProviderLookups collects the providers serving the connection
// types of the given inventory's assets.
func connectionProviderLookups(inv *inventory.Inventory) []providers.ProviderLookup {
	if inv == nil || inv.Spec == nil {
		return nil
	}

	seen := map[string]struct{}{}
	var lookups []providers.ProviderLookup
	for _, asset := range inv.Spec.Assets {
		if asset == nil {
			continue
		}
		for _, conn := range asset.Connections {
			if conn == nil || conn.Type == "" {
				continue
			}
			if _, ok := seen[conn.Type]; ok {
				continue
			}
			seen[conn.Type] = struct{}{}
			lookups = append(lookups, providers.ProviderLookup{ConnType: conn.Type})
		}
	}
	return lookups
}

// ensureConnectionProviders makes sure the providers that connect to the
// given inventory's assets are fully installed before discovery connects
// them. Bundle.EnsureRequirements only pulls schemas, and the runtime's
// provider lookup at connect time treats a schema-only install as installed
// even though it cannot be started.
func ensureConnectionProviders(inv *inventory.Inventory) error {
	return ensureProviderBinaries(connectionProviderLookups(inv))
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
