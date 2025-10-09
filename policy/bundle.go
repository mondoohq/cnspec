// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12"
	"go.mondoo.com/cnquery/v12/checksums"
	"go.mondoo.com/cnquery/v12/explorer"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/logger"
	"go.mondoo.com/cnquery/v12/mqlc"
	"go.mondoo.com/cnquery/v12/mrn"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v12/utils/multierr"
	"sigs.k8s.io/yaml"
)

const (
	MRN_RESOURCE_QUERY        = "queries"
	MRN_RESOURCE_POLICY       = "policies"
	MRN_RESOURCE_ASSET        = "assets"
	MRN_RESOURCE_FRAMEWORK    = "frameworks"
	MRN_RESOURCE_FRAMEWORKMAP = "frameworkmaps"
	MRN_RESOURCE_CONTROL      = "controls"
	MRN_RESOURCE_RISK         = "risks"
	MRN_RESOURCE_PROPERTY     = "properties"
)

type BundleResolver interface {
	Load(ctx context.Context, path string) (*Bundle, error)
	IsApplicable(path string) bool
}

type BundleLoader struct {
	resolvers []BundleResolver
}

func NewBundleLoader(resolvers ...BundleResolver) *BundleLoader {
	return &BundleLoader{resolvers: resolvers}
}

func DefaultBundleLoader() *BundleLoader {
	return NewBundleLoader(defaultS3BundleResolver(), defaultFileBundleResolver())
}

func (l *BundleLoader) getResolver(path string) (BundleResolver, error) {
	for _, resolver := range l.resolvers {
		if resolver.IsApplicable(path) {
			return resolver, nil
		}
	}
	return nil, fmt.Errorf("no resolver found for path '%s'", path)
}

// Deprecated: Use BundleLoader.BundleFromPaths instead
func BundleFromPaths(paths ...string) (*Bundle, error) {
	defaultLoader := DefaultBundleLoader()
	return defaultLoader.BundleFromPaths(paths...)
}

// iterates through all the resolvers until it finds an applicable one and then uses that to load the bundle
// from the provided path
func (l *BundleLoader) BundleFromPaths(paths ...string) (*Bundle, error) {
	ctx := context.Background()
	aggregatedBundle := &Bundle{}
	for _, path := range paths {
		resolver, err := l.getResolver(path)
		if err != nil {
			return nil, err
		}
		bundle, err := resolver.Load(ctx, path)
		if err != nil {
			log.Error().Err(err).Msg("could not resolve bundle files")
			return nil, err
		}
		aggregatedBundle = Merge(aggregatedBundle, bundle)
	}

	logger.DebugDumpYAML("resolved_mql_bundle.mql", aggregatedBundle)
	return aggregatedBundle, nil
}

// BundleExecutionChecksum creates a combined execution checksum from a policy
// and framework. Either may be nil.
func BundleExecutionChecksum(ctx context.Context, policy *Policy, framework *Framework) string {
	res := checksums.New
	if policy != nil {
		res = res.Add(policy.GraphExecutionChecksum)
	}
	if framework != nil {
		res = res.Add(framework.GraphExecutionChecksum)
	}
	// So far the checksum only includes the policy and the framework
	// It does not change if any of the jobs changes, only if the policy or the framework changes
	// To update the resolved policy, when we change how it is generated, change the incorporated version of the resolver
	res = res.Add(RESOLVER_VERSION)

	return res.String()
}

// Merge combines two PolicyBundle and merges the data additive into one
// single PolicyBundle structure
func Merge(a *Bundle, b *Bundle) *Bundle {
	res := &Bundle{}

	res.OwnerMrn = a.OwnerMrn
	if b.OwnerMrn != "" {
		res.OwnerMrn = b.OwnerMrn
	}

	// merge in a
	res.Policies = append(res.Policies, a.Policies...)
	res.Packs = append(res.Packs, a.Packs...)
	res.Props = append(res.Props, a.Props...)
	res.Queries = append(res.Queries, a.Queries...)
	res.Frameworks = append(res.Frameworks, a.Frameworks...)
	res.FrameworkMaps = append(res.FrameworkMaps, a.FrameworkMaps...)
	res.MigrationGroups = append(res.MigrationGroups, a.MigrationGroups...)

	// merge in b
	res.Policies = append(res.Policies, b.Policies...)
	res.Packs = append(res.Packs, b.Packs...)
	res.Props = append(res.Props, b.Props...)
	res.Queries = append(res.Queries, b.Queries...)
	res.Frameworks = append(res.Frameworks, b.Frameworks...)
	res.FrameworkMaps = append(res.FrameworkMaps, b.FrameworkMaps...)
	res.MigrationGroups = append(res.MigrationGroups, b.MigrationGroups...)

	return res
}

// BundleFromYAML create a policy bundle from yaml contents
func BundleFromYAML(data []byte) (*Bundle, error) {
	var res Bundle
	err := yaml.Unmarshal(data, &res)
	return &res, err
}

// ToYAML returns the policy bundle as yaml
func (p *Bundle) ToYAML() ([]byte, error) {
	return yaml.Marshal(p)
}

// Prepares the bundle for compilation
func (b *Bundle) Prepare() {
	b.ConvertEvidence()
	b.ConvertQuerypacks()
}

// ConvertQuerypacks takes any existing querypacks in the bundle
// and turns them into policies for execution by cnspec.
func (p *Bundle) ConvertQuerypacks() {
	for i := range p.Packs {
		pack := p.Packs[i]

		policy := Policy{
			Mrn:      pack.Mrn,
			Uid:      pack.Uid,
			Name:     pack.Name,
			Version:  pack.Version,
			License:  pack.License,
			OwnerMrn: pack.OwnerMrn,
			Docs:     convertQueryPackDocs(pack.Docs),
			Summary:  pack.Summary,
			Authors:  pack.Authors,
			Created:  pack.Created,
			Modified: pack.Modified,
			Tags:     pack.Tags,
			Props:    pack.Props,
			Groups:   convertQueryPackGroups(pack),
			// we need this to indicate that the policy was converted from a querypack
			ScoringSystem: explorer.ScoringSystem_DATA_ONLY,
		}
		p.Policies = append(p.Policies, &policy)
	}
}

func (b *Bundle) ConvertEvidence() {
	for _, f := range b.Frameworks {
		pol, fm := f.GenerateEvidenceObjects()
		if pol != nil {
			b.Policies = append(b.Policies, pol)
		}
		if fm != nil {
			b.FrameworkMaps = append(b.FrameworkMaps, fm)
		}
	}
}

func convertQueryPackDocs(q *explorer.QueryPackDocs) *PolicyDocs {
	if q == nil {
		return nil
	}
	return &PolicyDocs{
		Desc: q.Desc,
	}
}

func convertQueryPackGroups(p *explorer.QueryPack) []*PolicyGroup {
	var res []*PolicyGroup

	if len(p.Queries) > 0 {
		// any builtin queries need to be put into a group for policies
		res = append(res, &PolicyGroup{
			Queries: p.Queries,
			Type:    GroupType_CHAPTER,
			Uid:     "default-queries",
			Title:   "Default Queries",
			Filters: p.Filters,
		})
	}
	for i := range p.Groups {
		g := p.Groups[i]
		res = append(res, &PolicyGroup{
			Queries:  g.Queries,
			Type:     GroupType_CHAPTER,
			Filters:  g.Filters,
			Created:  g.Created,
			Modified: g.Modified,
			Title:    g.Title,
		})
	}

	return res
}

func (p *Bundle) SourceHash() (string, error) {
	raw, err := p.ToYAML()
	if err != nil {
		return "", err
	}
	c := checksums.New
	c = c.Add(string(raw))
	return c.String(), nil
}

// ToMap turns the PolicyBundle into a PolicyBundleMap
// dataLake (optional) may be used to provide queries/policies not found in the bundle
func (p *Bundle) ToMap() *PolicyBundleMap {
	res := NewPolicyBundleMap(p.OwnerMrn)

	for i := range p.Policies {
		c := p.Policies[i]
		res.Policies[c.Mrn] = c

		for j := range c.RiskFactors {
			r := c.RiskFactors[j]
			res.RiskFactors[r.Mrn] = r
		}
	}

	for i := range p.Frameworks {
		c := p.Frameworks[i]
		res.Frameworks[c.Mrn] = c
	}

	for i := range p.Queries {
		c := p.Queries[i]
		res.Queries[c.Mrn] = c
	}

	for i := range p.Props {
		c := p.Props[i]
		res.Props[c.Mrn] = c
	}

	return res
}

// FilterPolicies only keeps the given policy UIDs or MRNs and removes every other one.
// If a given policy has a MRN set (but no UID) it will try to get the UID from the MRN
// and also filter by that criteria.
// If the list of IDs is empty this function doesn't do anything.
// This function does not remove orphaned queries from the bundle.
func (p *Bundle) FilterPolicies(IDs []string) {
	if p == nil || len(IDs) == 0 {
		return
	}

	log.Debug().Msg("filter policies for asset")
	valid := make(map[string]struct{}, len(IDs))
	for i := range IDs {
		valid[IDs[i]] = struct{}{}
	}

	var cur *Policy
	var res []*Policy
	for i := range p.Policies {
		cur = p.Policies[i]

		if cur.Mrn != "" {
			if _, ok := valid[cur.Mrn]; ok {
				res = append(res, cur)
				continue
			}

			uid, _ := mrn.GetResource(cur.Mrn, MRN_RESOURCE_POLICY)
			if _, ok := valid[uid]; ok {
				res = append(res, cur)
				continue
			}

			log.Debug().Str("policy", cur.Mrn).Msg("policy does not match user-provided filter")
			// if we have a MRN we do not check the UID
			continue
		}

		if _, ok := valid[cur.Uid]; ok {
			res = append(res, cur)
			continue
		}
		log.Debug().Str("policy", cur.Uid).Msg("policy does not match user-provided filter")
	}

	p.Policies = res
}

func (p *Bundle) RemoveOrphaned() {
	panic("Not yet implemented, please open an issue at https://github.com/mondoohq/cnspec")
}

// Clean the policy bundle to turn a few nil fields into empty fields for consistency
func (p *Bundle) Clean() *Bundle {
	for i := range p.Policies {
		policy := p.Policies[i]
		if policy.ComputedFilters == nil {
			policy.ComputedFilters = &explorer.Filters{
				Items: map[string]*explorer.Mquery{},
			}
		}
	}

	// consistency between db backends
	if p.Props != nil && len(p.Props) == 0 {
		p.Props = nil
	}

	return p
}

// Add another policy bundle into this. No duplicate policies, queries, or
// properties are allowed and will lead to an error. Both bundles must have
// MRNs for everything. OwnerMRNs must be identical as well.
func (p *Bundle) AddBundle(other *Bundle) error {
	if p.OwnerMrn == "" {
		p.OwnerMrn = other.OwnerMrn
	} else if p.OwnerMrn != other.OwnerMrn {
		return errors.New("when combining policy bundles the owner MRNs must be identical")
	}

	for i := range other.Policies {
		c := other.Policies[i]
		if c.Mrn == "" {
			return errors.New("source policy bundle that is added has missing policy MRNs")
		}

		for j := range p.Policies {
			if p.Policies[j].Mrn == c.Mrn {
				return errors.New("cannot combine policy bundles, duplicate policy: " + c.Mrn)
			}
		}

		p.Policies = append(p.Policies, c)
	}

	for i := range other.Queries {
		c := other.Queries[i]
		if c.Mrn == "" {
			return errors.New("source policy bundle that is added has missing query MRNs")
		}

		for j := range p.Queries {
			if p.Queries[j].Mrn == c.Mrn {
				return errors.New("cannot combine policy bundles, duplicate query: " + c.Mrn)
			}
		}

		p.Queries = append(p.Queries, c)
	}

	for i := range other.Props {
		c := other.Props[i]
		if c.Mrn == "" {
			return errors.New("source policy bundle that is added has missing property MRNs")
		}

		for j := range p.Props {
			if p.Props[j].Mrn == c.Mrn {
				return errors.New("cannot combine policy bundles, duplicate property: " + c.Mrn)
			}
		}

		p.Props = append(p.Props, c)
	}

	return nil
}

// PolicyMRNs in this bundle
func (p *Bundle) PolicyMRNs() []string {
	mrns := []string{}
	for i := range p.Policies {
		// ensure a mrn is generated
		p.Policies[i].RefreshMRN(p.OwnerMrn)
		mrns = append(mrns, p.Policies[i].Mrn)
	}
	return mrns
}

// Sorts the queries, policies and queries' variants in the bundle.
func (p *Bundle) SortContents() {
	sort.SliceStable(p.Queries, func(i, j int) bool {
		if p.Queries[i].Mrn == "" || p.Queries[j].Mrn == "" {
			return p.Queries[i].Uid < p.Queries[j].Uid
		}
		return p.Queries[i].Mrn < p.Queries[j].Mrn
	})

	sort.SliceStable(p.Policies, func(i, j int) bool {
		if p.Policies[i].Mrn == "" || p.Policies[j].Mrn == "" {
			return p.Policies[i].Uid < p.Policies[j].Uid
		}
		return p.Policies[i].Mrn < p.Policies[j].Mrn
	})

	for _, q := range p.Queries {
		sort.SliceStable(q.Variants, func(i, j int) bool {
			if q.Variants[i].Mrn == "" || q.Variants[j].Mrn == "" {
				return q.Variants[i].Uid < q.Variants[j].Uid
			}
			return q.Variants[i].Mrn < q.Variants[j].Mrn
		})
	}
	for _, pl := range p.Policies {
		for _, g := range pl.Groups {
			for _, q := range g.Queries {
				sort.SliceStable(q.Variants, func(i, j int) bool {
					if q.Variants[i].Mrn == "" || q.Variants[j].Mrn == "" {
						return q.Variants[i].Uid < q.Variants[j].Uid
					}
					return q.Variants[i].Mrn < q.Variants[j].Mrn
				})
			}
			for _, c := range g.Checks {
				sort.SliceStable(c.Variants, func(i, j int) bool {
					if c.Variants[i].Mrn == "" || c.Variants[j].Mrn == "" {
						return c.Variants[i].Uid < c.Variants[j].Uid
					}
					return c.Variants[i].Mrn < c.Variants[j].Mrn
				})
			}
		}
	}
}

type docsPrinter interface {
	Write(section string, data string)
}

type nocodeWriter struct {
	docsWriter
}

var reCode = regexp.MustCompile("(?s)`[^`]+`|```.*```")

func (n nocodeWriter) Write(section string, data string) {
	n.docsWriter.Write(section, reCode.ReplaceAllString(data, ""))
}

type docsWriter struct {
	out io.Writer
}

func (n docsWriter) Write(section string, data string) {
	if data == "" {
		return
	}
	io.WriteString(n.out, section)
	io.WriteString(n.out, ": ")
	io.WriteString(n.out, data)
	n.out.Write([]byte{'\n'})
}

func extractQueryDocs(query *explorer.Mquery, w docsPrinter, noIDs bool) {
	if !noIDs {
		w.Write("query ID", query.Uid)
		w.Write("query Mrn", query.Mrn)
	}

	w.Write("query title", query.Title)
	w.Write("query description", query.Desc)

	if query.Docs != nil {
		w.Write("query description", query.Docs.Desc)
		w.Write("query audit", query.Docs.Audit)
		if query.Docs.Remediation != nil {
			for i := range query.Docs.Remediation.Items {
				remediation := query.Docs.Remediation.Items[i]
				if noIDs {
					w.Write("query remediation", remediation.Desc)
				} else {
					w.Write("query remediation "+remediation.Id, remediation.Desc)
				}
			}
		}
	}
}

func extractPropertyDocs(prop *explorer.Property, w docsPrinter, noIDs bool) {
	if !noIDs {
		w.Write("property ID", prop.Uid)
		w.Write("property Mrn", prop.Mrn)
	}

	w.Write("property title", prop.Title)
	w.Write("property description", prop.Desc)
}

func extractPolicyDocs(policy *Policy, w docsPrinter, noIDs bool) {
	if !noIDs {
		w.Write("policy ID", policy.Uid)
		w.Write("policy Mrn", policy.Mrn)
	}

	w.Write("policy summary", policy.Summary)

	for i := range policy.Props {
		extractPropertyDocs(policy.Props[i], w, noIDs)
	}
	if policy.Docs != nil {
		w.Write("policy description", policy.Docs.Desc)
	}

	for g := range policy.Groups {
		group := policy.Groups[g]

		w.Write("group summary", group.Title)

		if group.Docs != nil {
			w.Write("group description", group.Docs.Desc)
		}

		for i := range group.Queries {
			extractQueryDocs(group.Queries[i], w, noIDs)
		}
		for i := range group.Checks {
			extractQueryDocs(group.Checks[i], w, noIDs)
		}
	}
}

func (p *Bundle) ExtractDocs(out io.Writer, noIDs bool, noCode bool) {
	if p == nil {
		return
	}

	var printer docsPrinter
	if noCode {
		printer = nocodeWriter{docsWriter: docsWriter{out}}
	} else {
		printer = docsWriter{out}
	}

	for i := range p.Queries {
		extractQueryDocs(p.Queries[i], printer, noIDs)
	}
	for i := range p.Props {
		extractPropertyDocs(p.Props[i], printer, noIDs)
	}
	for i := range p.Policies {
		extractPolicyDocs(p.Policies[i], printer, noIDs)
	}
}

func (p *bundleCache) ensureNoCyclesInVariants(queries []*explorer.Mquery) error {
	// Gather all top-level queries with variants
	queriesMap := map[string]*explorer.Mquery{}
	for _, q := range queries {
		if q == nil {
			continue
		}
		if q.Mrn == "" {
			// This should never happen. This function is called after all
			// queries have their MRNs set.
			panic("BUG: expected query MRN to be set for variant cycle detection")
		}
		queriesMap[q.Mrn] = q
	}

	if err := detectVariantCycles(queriesMap); err != nil {
		return err
	}
	return nil
}

func (c *bundleCache) removeFailing(res *Bundle) {
	if !c.conf.RemoveFailing {
		return
	}

	for k := range c.removeQueries {
		log.Debug().Str("query", k).Msg("removing query from bundle")
	}
	res.Queries = explorer.FilterQueryMRNs(c.removeQueries, res.Queries)

	for i := range res.Policies {
		policy := res.Policies[i]

		groups := []*PolicyGroup{}
		for j := range policy.Groups {
			group := policy.Groups[j]
			group.Queries = explorer.FilterQueryMRNs(c.removeQueries, group.Queries)
			group.Checks = explorer.FilterQueryMRNs(c.removeQueries, group.Checks)
			if len(group.Policies)+len(group.Queries)+len(group.Checks) > 0 {
				groups = append(groups, group)
			}
		}

		policy.Groups = groups
	}
}

type nodeVisitStatus byte

const (
	// NEW is the initial state of visiting a node. It means that the node has not been visited yet.
	NEW nodeVisitStatus = iota
	// ACTIVE means that the node is currently being visited. If we encounter a node that is in
	// ACTIVE state, it means that we have a cycle.
	ACTIVE
	// VISITED means that the node has been visited.
	VISITED
)

var ErrVariantCycleDetected = func(mrn string) error {
	return fmt.Errorf("variant cycle detected in %s", mrn)
}

func detectVariantCycles(queries map[string]*explorer.Mquery) error {
	statusMap := map[string]nodeVisitStatus{}
	for _, query := range queries {
		err := detectVariantCyclesDFS(query.Mrn, statusMap, queries)
		if err != nil {
			return err
		}
	}
	return nil
}

func detectVariantCyclesDFS(mrn string, statusMap map[string]nodeVisitStatus, queries map[string]*explorer.Mquery) error {
	q := queries[mrn]
	if q == nil {
		return nil
	}
	s := statusMap[mrn]
	if s == VISITED {
		return nil
	} else if s == ACTIVE {
		return ErrVariantCycleDetected(mrn)
	}
	statusMap[q.Mrn] = ACTIVE
	for _, variant := range q.Variants {
		if variant.Mrn == "" {
			// This should never happen. This function is called after all
			// queries have their MRNs set.
			panic("BUG: expected variant MRN to be set for variant cycle detection")
		}
		v := queries[variant.Mrn]
		if v == nil {
			continue
		}
		err := detectVariantCyclesDFS(v.Mrn, statusMap, queries)
		if err != nil {
			return err
		}
	}
	statusMap[q.Mrn] = VISITED
	return nil
}

func topologicalSortQueries(queries []*explorer.Mquery) ([]*explorer.Mquery, error) {
	// Gather all top-level queries with variants
	queriesMap := map[string]*explorer.Mquery{}
	for _, q := range queries {
		if q == nil {
			continue
		}
		if q.Mrn == "" {
			// This should never happen. This function is called after all
			// queries have their MRNs set.
			panic("BUG: expected query MRN to be set for topological sort")
		}
		queriesMap[q.Mrn] = q
	}

	// Topologically sort the queries
	sorted := &explorer.Mqueries{}
	visited := map[string]struct{}{}
	for _, q := range queriesMap {
		err := topologicalSortQueriesDFS(q.Mrn, queriesMap, visited, sorted)
		if err != nil {
			return nil, err
		}
	}

	return sorted.Items, nil
}

func topologicalSortQueriesDFS(queryMrn string, queriesMap map[string]*explorer.Mquery, visited map[string]struct{}, sorted *explorer.Mqueries) error {
	if _, ok := visited[queryMrn]; ok {
		return nil
	}
	visited[queryMrn] = struct{}{}
	q := queriesMap[queryMrn]
	if q == nil {
		return nil
	}
	for _, variant := range q.Variants {
		if variant.Mrn == "" {
			// This should never happen. This function is called after all
			// queries have their MRNs set.
			panic("BUG: expected variant MRN to be set for topological sort")
		}
		err := topologicalSortQueriesDFS(variant.Mrn, queriesMap, visited, sorted)
		if err != nil {
			return err
		}
	}
	sorted.Items = append(sorted.Items, q)
	return nil
}

// Compile a bundle. See CompileExt for a full description.
func (p *Bundle) Compile(ctx context.Context, schema resources.ResourcesSchema, library Library) (*PolicyBundleMap, error) {
	return p.CompileExt(ctx, BundleCompileConf{
		CompilerConfig: mqlc.NewConfig(schema, cnquery.DefaultFeatures),
		Library:        library,
	})
}

type BundleCompileConf struct {
	mqlc.CompilerConfig
	Library       Library
	RemoveFailing bool
}

// Compile PolicyBundle into a PolicyBundleMap
// Does 4 things:
// 1. turns policy bundle into a map for easier access
// 2. compile all queries. store code in the bundle map
// 3. validation of all contents
// 4. generate MRNs for all policies, queries, and properties and updates referencing local fields
// 5. snapshot all queries into the packs
// 6. make queries public that are only embedded
func (p *Bundle) CompileExt(ctx context.Context, conf BundleCompileConf) (*PolicyBundleMap, error) {
	ownerMrn := p.OwnerMrn
	if ownerMrn == "" {
		// this only happens for local bundles where queries have no mrn yet
		ownerMrn = "//local.cnspec.io/run/local-execution"
	}

	cache := &bundleCache{
		ownerMrn:      ownerMrn,
		bundle:        p,
		conf:          conf,
		uid2mrn:       map[string]string{},
		removeQueries: map[string]struct{}{},
		lookupProps:   map[string]explorer.PropertyRef{},
		lookupQuery:   map[string]*explorer.Mquery{},
		codeBundles:   map[string]*llx.CodeBundle{},
		parents:       map[string][]parent{},
	}

	// Process variants and inherit attributes filled from their parents
	for _, query := range p.Queries {
		if len(query.Variants) == 0 {
			continue
		}
		// we do not have a bundle map yet. we need to do the check here
		// so props are copied down before we compile props and queries
		for i := range query.Variants {
			ref := query.Variants[i]

			for _, variant := range p.Queries {
				if (variant.Uid != "" && variant.Uid == ref.Uid) ||
					(variant.Mrn != "" && variant.Mrn == ref.Mrn) {
					addBaseToVariant(query, variant)
				}
			}
		}
	}

	if err := cache.prepareMRNs(); err != nil {
		return nil, err
	}

	if err := cache.buildParents(); err != nil {
		return nil, err
	}

	if err := cache.compileQueries(p.Queries, nil); err != nil {
		return nil, err
	}

	// Index policies + update MRNs and checksums, link properties via MRNs
	for i := range p.Policies {
		policy := p.Policies[i]
		policyCache := cache.clone()

		// !this is very important to prevent user overrides! vv
		policy.InvalidateAllChecksums()

		for i := range policy.Props {
			np, ok := policyCache.lookupProps[policy.Props[i].Uid]
			if ok {
				policy.Props[i] = np.Property
			}
		}

		// Filters: prep a data structure in case it doesn't exist yet and add
		// any filters that child groups may carry with them
		if policy.ComputedFilters == nil || policy.ComputedFilters.Items == nil {
			policy.ComputedFilters = &explorer.Filters{Items: map[string]*explorer.Mquery{}}
		}
		if err := policy.ComputedFilters.Compile(ownerMrn, conf.CompilerConfig); err != nil {
			return nil, multierr.Wrap(err, "failed to compile policy filters")
		}

		// ---- GROUPs -------------
		for i := range policy.Groups {
			group := policy.Groups[i]

			// When filters are initially added they haven't been compiled
			if err := group.Filters.Compile(ownerMrn, conf.CompilerConfig); err != nil {
				return nil, multierr.Wrap(err, "failed to compile policy group filters")
			}

			if group.Filters != nil {
				for j := range group.Filters.Items {
					filter := group.Filters.Items[j]
					policy.ComputedFilters.Items[filter.CodeId] = filter
				}
			}

			for j := range group.Policies {
				policyRef := group.Policies[j]
				if err := policyRef.RefreshMRN(ownerMrn); err != nil {
					return nil, err
				}
				policyRef.RefreshChecksum()
			}

			if err := policyCache.compileQueries(group.Queries, policy); err != nil {
				return nil, err
			}
			if err := policyCache.compileQueries(group.Checks, policy); err != nil {
				return nil, err
			}
		}

		// ---- RISK FACTORS ---------
		for i := range policy.RiskFactors {
			risk := policy.RiskFactors[i]
			if err := risk.RefreshMRN(ownerMrn); err != nil {
				return nil, errors.New("failed to assign MRN to risk: " + err.Error())
			}

			risk.DetectScope()

			if err := policyCache.compileRisk(risk, policy); err != nil {
				return nil, errors.New("failed to compile risk: " + err.Error())
			}
		}

		policyCache.finalize(cache)
	}

	// Removing any failing queries happens after everything is compiled.
	// We do this to the original bundle, because the intent is to
	// clean it up with this option.
	cache.removeFailing(p)

	frameworksByMrn := map[string]*Framework{}
	for _, framework := range p.Frameworks {
		if err := framework.compile(ctx, ownerMrn, cache); err != nil {
			return nil, errors.New("failed to validate framework: " + err.Error())
		}
		frameworksByMrn[framework.Mrn] = framework
	}

	for i := range p.FrameworkMaps {
		fm := p.FrameworkMaps[i]
		if err := fm.compile(ctx, ownerMrn, cache); err != nil {
			return nil, errors.New("failed to validate framework map: " + err.Error())
		}

		framework, ok := frameworksByMrn[fm.FrameworkOwner.Mrn]
		if !ok {
			return nil, errors.New("failed to get framework in bundle (not yet supported) for " + fm.FrameworkOwner.Mrn)
		}
		framework.FrameworkMaps = append(framework.FrameworkMaps, fm)
	}

	// cannot be done before all policies and queries have their MRNs set
	bundleMap := p.ToMap()
	bundleMap.Library = cache.conf.Library
	bundleMap.Code = cache.codeBundles

	// Validate integrity of references + translate UIDs to MRNs
	for i := range p.Policies {
		policy := p.Policies[i]
		if policy == nil {
			return nil, errors.New("received null policy")
		}

		err := translateGroupUIDs(ownerMrn, policy, cache.uid2mrn)
		if err != nil {
			return nil, errors.New("failed to validate policy: " + err.Error())
		}

		err = bundleMap.ValidatePolicy(ctx, policy, cache.conf.CompilerConfig)
		if err != nil {
			return nil, errors.New("failed to validate policy: " + err.Error())
		}
	}

	return bundleMap, cache.error()
}

func LiftPropertiesToPolicy(policy *Policy, lookupQuery map[string]*explorer.Mquery) error {
	if len(policy.Props) != 0 {
		// If these properties are defined by uid, we need to lift the MRNs
		// into the for field of the property.
		propsByUid := map[string][]string{}
		for _, g := range policy.Groups {
			for _, arr := range [][]*explorer.Mquery{g.Queries, g.Checks} {
				for _, q := range arr {
					resolvedQuery := lookupQuery[q.Mrn]
					if resolvedQuery == nil {
						return fmt.Errorf("failed to resolve query %s in policy %s", q.Mrn, policy.Mrn)
					}
					for _, prop := range resolvedQuery.Props {
						propUid, err := explorer.GetPropName(prop.Mrn)
						if err != nil {
							return fmt.Errorf("failed to get property name for property %s in policy %s: %w", prop.Mrn, policy.Mrn, err)
						}
						propsByUid[propUid] = append(propsByUid[propUid], prop.Mrn)
					}
				}
			}
		}
		for _, prop := range policy.Props {
			if len(prop.For) == 0 {
				continue
			}
			newFor := []*explorer.ObjectRef{}
			for _, pFor := range prop.For {
				if pFor.Mrn != "" {
					newFor = append(newFor, pFor)
					continue
				}

				mrns := propsByUid[pFor.Uid]
				for _, mrn := range mrns {
					newFor = append(newFor, &explorer.ObjectRef{
						Mrn: mrn,
					})
				}
			}
			prop.For = newFor
		}
		return nil
	}
	newPolicyProps := map[string]*explorer.Property{}
	for _, g := range policy.Groups {
		for _, arr := range [][]*explorer.Mquery{g.Queries, g.Checks} {
			for _, q := range arr {
				resolvedQuery := lookupQuery[q.Mrn]
				if resolvedQuery == nil {
					return fmt.Errorf("failed to resolve query %s in policy %s", q.Mrn, policy.Mrn)
				}
				if len(resolvedQuery.Props) == 0 {
					continue
				}
				for _, prop := range resolvedQuery.Props {
					propUid, err := explorer.GetPropName(prop.Mrn)
					if err != nil {
						return fmt.Errorf("failed to get property name for query %s in policy %s: %w", resolvedQuery.Mrn, policy.Mrn, err)
					}

					policyProp := newPolicyProps[propUid]
					if policyProp == nil {
						policyProp = &explorer.Property{
							Uid:  propUid,
							Type: prop.Type,
						}
						newPolicyProps[propUid] = policyProp
					}
					policyProp.For = append(policyProp.For, &explorer.ObjectRef{
						Mrn: prop.Mrn,
					})
				}
			}
		}
	}
	keys := make([]string, 0, len(newPolicyProps))
	for k := range newPolicyProps {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		prop := newPolicyProps[k]
		if err := prop.RefreshMRN(policy.Mrn); err != nil {
			return fmt.Errorf("failed to refresh MRN for property %s in policy %s: %w", prop.Mrn, policy.Mrn, err)
		}
		policy.Props = append(policy.Props, prop)
	}
	return nil
}

// this uses a subset of the calls of Mquery.AddBase(), because we don't want
// to push all the fields into the variant, only a select few that are
// needed for context and execution.
func addBaseToVariant(base *explorer.Mquery, variant *explorer.Mquery) {
	if variant == nil {
		return
	}

	if variant.Title == "" {
		variant.Title = base.Title
	}
	if variant.Desc == "" {
		variant.Desc = base.Desc
	}
	if variant.Docs == nil {
		variant.Docs = base.Docs
	} else if base.Docs != nil {
		if variant.Docs.Desc == "" {
			variant.Docs.Desc = base.Docs.Desc
		}
		if variant.Docs.Audit == "" {
			variant.Docs.Audit = base.Docs.Audit
		}
		if variant.Docs.Remediation == nil {
			variant.Docs.Remediation = base.Docs.Remediation
		}
		if variant.Docs.Refs == nil {
			variant.Docs.Refs = base.Docs.Refs
		}
	}
	if variant.Impact == nil {
		variant.Impact = base.Impact
	}
	if len(variant.Props) == 0 {
		variant.Props = base.Props
	}

	// We are not copying filters, variants should have their own.

	// We can't copy Tags because the parent query can have way more tags
	// than are applicable to the variant.
}

type parent struct {
	policy *Policy
	query  *explorer.Mquery
}

type bundleCache struct {
	ownerMrn      string
	lookupQuery   map[string]*explorer.Mquery
	lookupProps   map[string]explorer.PropertyRef
	uid2mrn       map[string]string
	removeQueries map[string]struct{}
	codeBundles   map[string]*llx.CodeBundle
	bundle        *Bundle
	conf          BundleCompileConf
	errors        []error
	parents       map[string][]parent
}

func (c *bundleCache) clone() *bundleCache {
	res := &bundleCache{
		ownerMrn:      c.ownerMrn,
		lookupQuery:   make(map[string]*explorer.Mquery, len(c.lookupQuery)),
		lookupProps:   make(map[string]explorer.PropertyRef, len(c.lookupProps)),
		uid2mrn:       make(map[string]string, len(c.uid2mrn)),
		removeQueries: c.removeQueries,
		bundle:        c.bundle,
		errors:        c.errors,
		conf:          c.conf,
	}
	for k, v := range c.lookupQuery {
		res.lookupQuery[k] = v
	}
	for k, v := range c.lookupProps {
		res.lookupProps[k] = v
	}
	for k, v := range c.uid2mrn {
		res.uid2mrn[k] = v
	}
	return res
}

func (c *bundleCache) finalize(parent *bundleCache) {
	parent.errors = append(parent.errors, c.errors...)
}

func (c *bundleCache) hasErrors() bool {
	return len(c.errors) != 0
}

func (c *bundleCache) error() error {
	if len(c.errors) == 0 {
		return nil
	}

	var msg strings.Builder
	for i := range c.errors {
		msg.WriteString(c.errors[i].Error())
		msg.WriteString("\n")
	}
	return errors.New(msg.String())
}

// prepareMRNs is responsible for turning UIDs into MRNs. It also does the
// lifting of properties to the policy level
func (cache *bundleCache) prepareMRNs() error {
	refreshPropMrns := func(props []*explorer.Property, ownerMrn string) error {
		for _, prop := range props {
			var name string

			if prop.Mrn == "" {
				uid := prop.Uid
				if err := prop.RefreshMRN(ownerMrn); err != nil {
					return err
				}

				// TODO: uid's can be namespaced, extract the name
				name = uid
			} else {
				m, err := mrn.NewMRN(prop.Mrn)
				if err != nil {
					return errors.Wrap(err, "failed to compile prop, invalid mrn: "+prop.Mrn)
				}

				name = m.Basename()
			}

			_, ok := cache.lookupProps[prop.Mrn]
			if ok && prop.Mql == "" {
				// this is a shared property, we can skip it
				// TODO: this can go away. Props are now scoped and thus
				// cannot interfere with each other.
				continue
			}

			cache.lookupProps[prop.Mrn] = explorer.PropertyRef{
				Property: prop,
				Name:     name,
			}
		}

		return nil
	}

	refreshQueryMrns := func(queries []*explorer.Mquery, ownerMrn string) error {
		for _, query := range queries {
			uid := query.Uid
			if err := query.RefreshMRN(cache.ownerMrn); err != nil {
				return fmt.Errorf("failed to refresh MRN for query %s: %w", query.Uid, err)
			}
			if uid != "" {
				cache.uid2mrn[uid] = query.Mrn
			}

			for i := range query.Variants {
				variant := query.Variants[i]
				uid := variant.Uid
				if err := variant.RefreshMRN(cache.ownerMrn); err != nil {
					return errors.New("failed to refresh MRN for variant in query " + query.Uid)
				}
				if uid != "" {
					cache.uid2mrn[uid] = variant.Mrn
				}
			}

			// ensure MRNs for properties
			if err := refreshPropMrns(query.Props, query.Mrn); err != nil {
				return fmt.Errorf("failed to refresh MRNs for properties in query %s: %w", query.Mrn, err)
			}
		}

		return nil
	}

	fillOutLookupQuery := func(queries []*explorer.Mquery, policy *Policy) {
		for _, query := range queries {
			if query.Mql == "" {
				// Query is deprecated. Calling compile does this migration, but we're
				// potentially cloning and merging queries here, and no compilation
				// has been done yet.
				query.Mql = query.Query
			}
			// the policy is only nil if we are dealing with shared queries
			if policy == nil {
				cache.lookupQuery[query.Mrn] = query
			} else if existing, ok := cache.lookupQuery[query.Mrn]; ok {
				query = query.Merge(existing)
				cache.lookupQuery[query.Mrn] = query
			} else {
				// Any other query that is in a pack, that does not exist globally,
				// we share out to be available in the bundle.
				cache.bundle.Queries = append(cache.bundle.Queries, query)
				cache.lookupQuery[query.Mrn] = query
			}
		}
	}

	if err := refreshQueryMrns(cache.bundle.Queries, cache.ownerMrn); err != nil {
		return fmt.Errorf("failed to refresh MRNs for queries: %w", err)
	}
	fillOutLookupQuery(cache.bundle.Queries, nil)

	for _, policy := range cache.bundle.Policies {
		// make sure we get a copy of the UID before it is removed (via refresh MRN)
		policyUID := policy.Uid

		if err := policy.RefreshMRN(cache.ownerMrn); err != nil {
			return fmt.Errorf("failed to refresh MRN for policy %s: %w", policy.Uid, err)
		}

		if policyUID != "" {
			cache.uid2mrn[policyUID] = policy.Mrn
		}

		// ensure MRNs for properties
		if err := refreshPropMrns(policy.Props, policy.Mrn); err != nil {
			return fmt.Errorf("failed to refresh MRNs for properties in policy %s: %w", policy.Mrn, err)
		}

		for _, group := range policy.Groups {
			// ensure MRNs for queries
			if err := refreshQueryMrns(group.Queries, cache.ownerMrn); err != nil {
				return fmt.Errorf("failed to refresh MRNs for queries in group %s: %w", group.Title, err)
			}
			fillOutLookupQuery(group.Queries, policy)

			if err := refreshQueryMrns(group.Checks, cache.ownerMrn); err != nil {
				return fmt.Errorf("failed to refresh MRNs for checks in group %s: %w", group.Title, err)
			}
			fillOutLookupQuery(group.Checks, policy)
		}

		if err := LiftPropertiesToPolicy(policy, cache.lookupQuery); err != nil {
			return fmt.Errorf("failed to lift properties to policy %s: %w", policy.Mrn, err)
		}
	}

	// ensure MRNs for properties
	if err := refreshPropMrns(cache.bundle.Props, cache.ownerMrn); err != nil {
		return fmt.Errorf("failed to refresh MRNs for properties in bundle: %w", err)
	}

	// We'll replace the properties that do not have an implementation
	replacePropIfNecessary := func(prop *explorer.Property) {
		for _, forProp := range prop.For {
			if existing, ok := cache.lookupProps[forProp.Mrn]; ok && existing.Property.Mql == "" {
				cache.lookupProps[forProp.Mrn] = explorer.PropertyRef{
					Property: prop,
					Name:     existing.Name,
				}
			}
		}
	}
	for _, prop := range cache.bundle.Props {
		replacePropIfNecessary(prop)
	}
	for _, policy := range cache.bundle.Policies {
		for _, prop := range policy.Props {
			replacePropIfNecessary(prop)
		}
	}

	// Compile the properties
	for _, prop := range cache.lookupProps {
		if prop.Property.Mql == "" {
			continue
		}
		if _, err := prop.RefreshChecksumAndType(cache.conf.CompilerConfig); err != nil {
			return err
		}
	}

	return nil
}

func (c *bundleCache) buildParents() error {
	for _, p := range c.bundle.Policies {
		for _, g := range p.Groups {
			for _, arr := range [][]*explorer.Mquery{g.Queries, g.Checks} {
				for _, q := range arr {
					if q == nil {
						continue
					}
					c.parents[q.Mrn] = append(c.parents[q.Mrn], parent{policy: p})
				}
			}
		}
	}

	for _, q := range c.bundle.Queries {
		for _, v := range q.Variants {
			if v == nil {
				continue
			}
			c.parents[v.Mrn] = append(c.parents[v.Mrn], parent{query: q})
		}
	}

	return nil
}

func (c *bundleCache) compileQueries(queries []*explorer.Mquery, policy *Policy) error {
	mergedQueries := make([]*explorer.Mquery, len(queries))
	for i := range queries {
		mergedQueries[i] = c.precompileQuery(queries[i], policy)
	}

	// Check for cycles in variants
	if err := c.ensureNoCyclesInVariants(mergedQueries); err != nil {
		return err
	}

	// Topologically sort the queries so that variant queries are compiled after the
	// actual query they include.
	topoSortedQueries, err := topologicalSortQueries(mergedQueries)
	if err != nil {
		return err
	}

	// After the first pass we may have errors. We try to collect as many errors
	// as we can before returning, so more problems can be fixed at once.
	// We have to return at this point, because these errors will prevent us from
	// compiling the queries.
	if c.hasErrors() {
		return c.error()
	}

	// Compile queries
	for _, m := range topoSortedQueries {
		c.compileQuery(m)
	}

	for i := range mergedQueries {
		query := mergedQueries[i]
		if query != nil {
			if query != queries[i] {
				queries[i].Checksum = query.Checksum
				queries[i].CodeId = query.CodeId
				queries[i].Type = query.Type
			}
		}
	}

	// The second pass on errors is done after we have compiled as much as possible.
	// Since shared queries may be used in other places, any errors here will prevent
	// us from compiling further.
	return c.error()
}

// precompileQuery indexes the query, turns UIDs into MRNs, compiles properties
// and filters, and pre-processes variants. Also makes sure the query isn't nil.
func (c *bundleCache) precompileQuery(query *explorer.Mquery, policy *Policy) *explorer.Mquery {
	if query == nil {
		c.errors = append(c.errors, errors.New("query or check is null"))
		return nil
	}

	if query.Title == "" {
		query.Title = query.Mql
	}

	// remove leading and trailing whitespace of docs, refs and tags
	query.Sanitize()

	queryMrn := query.Mrn
	query, ok := c.lookupQuery[query.Mrn]
	if !ok {
		// The query should be in the bundle
		c.errors = append(c.errors, fmt.Errorf("query %s not found in bundle", queryMrn))
		return nil
	}

	// filters have no dependencies, so we can compile them early
	if err := query.Filters.Compile(c.ownerMrn, c.conf.CompilerConfig); err != nil {
		c.errors = append(c.errors, errors.New("failed to compile filters for query "+query.Mrn))
		return nil
	}

	// ensure MRNs for variants

	// Filters will need to be aggregated into the pack's filters
	// note: must happen after all MRNs (including variants) are computed
	if policy != nil {
		if err := policy.ComputedFilters.AddQueryFilters(query, c.lookupQuery); err != nil {
			c.errors = append(c.errors, fmt.Errorf("failed to register filters for query %s: %v", query.Mrn, err))
			return nil
		}
	}

	return query
}

type QueryPropsResolver struct {
	query      *explorer.Mquery
	parents    map[string][]parent
	nameToProp map[string]*explorer.Property

	errors []error
}

func newQueryPropsResolver(query *explorer.Mquery, parents map[string][]parent) (*QueryPropsResolver, error) {
	propsMap := map[string]*explorer.Property{}
	for _, p := range query.Props {
		name, err := explorer.GetPropName(p.Mrn)
		if err != nil {
			return nil, err
		}
		propsMap[name] = p
	}
	return &QueryPropsResolver{
		query:      query,
		parents:    parents,
		nameToProp: propsMap,
		errors:     []error{},
	}, nil
}

func (r *QueryPropsResolver) Get(name string) *llx.Primitive {
	// Check explicitlyl defined properties
	queryProp, ok := r.nameToProp[name]
	if ok {
		if queryProp.Type == "" {
			// Resolve a type if the prop was defined with just a name
			r.walkParents(func(p hasProps) bool {
				for _, parentProp := range p.GetProps() {
					pn, err := explorer.GetPropName(parentProp.Mrn)
					if err != nil {
						continue
					}
					if pn == name && parentProp.Type != "" {
						queryProp.Type = parentProp.Type
						return true
					}
				}
				return false
			})
			if queryProp.Type == "" {
				r.errors = append(r.errors, errors.New("property "+name+" has no type in query "+r.query.Mrn))
				return nil
			}
		}
		return &llx.Primitive{Type: queryProp.Type}
	}

	found := struct {
		mrn  string
		prop *explorer.Property
	}{}
	r.walkParents(func(p hasProps) bool {
		for _, prop := range p.GetProps() {
			propName, err := explorer.GetPropName(prop.Mrn)
			if err != nil {
				continue
			}
			if propName == name {
				found.mrn = prop.Mrn
				found.prop = prop
				return true
			}
		}
		return false
	})

	if found.prop != nil {
		// Create an explicit property in the query for this implicit property.
		newProp := &explorer.Property{
			Uid:  name,
			Type: found.prop.Type,
		}
		if err := newProp.RefreshMRN(r.query.Mrn); err != nil {
			r.errors = append(r.errors, errors.New("failed to create MRN for implicit property "+name+" in query "+r.query.Mrn))
			return nil
		}
		r.query.Props = append(r.query.Props, newProp)
		// Reference the new property from the used property
		found.prop.For = append(found.prop.For, &explorer.ObjectRef{Mrn: newProp.Mrn})
		return &llx.Primitive{Type: found.prop.Type}
	}

	return nil
}

func (r *QueryPropsResolver) Available() map[string]*llx.Primitive {
	available := map[string]*llx.Primitive{}
	for name, prop := range r.nameToProp {
		available[name] = &llx.Primitive{Type: prop.Type}
	}
	return available
}

func (r *QueryPropsResolver) All() map[string]*llx.Primitive {
	all := map[string]*llx.Primitive{}
	for name, prop := range r.nameToProp {
		all[name] = &llx.Primitive{Type: prop.Type}
	}
	r.walkParents(func(p hasProps) bool {
		for _, prop := range p.GetProps() {
			propName, err := explorer.GetPropName(prop.Mrn)
			if err != nil {
				continue
			}
			if _, ok := all[propName]; !ok {
				all[propName] = &llx.Primitive{Type: prop.Type}
			}
		}
		return false
	})
	return all
}

type hasProps interface {
	GetMrn() string
	GetProps() []*explorer.Property
}

func (r *QueryPropsResolver) walkParents(f func(p hasProps) bool) {
	var walk func(parents []parent) bool
	walk = func(parents []parent) bool {
		for _, p := range parents {
			var hp hasProps
			if p.query != nil {
				hp = p.query
			} else if p.policy != nil {
				hp = p.policy
			}
			if hp == nil {
				continue
			}
			if f(hp) {
				return true
			}
			if walk(r.parents[hp.GetMrn()]) {
				return true
			}

		}
		return false
	}
	walk(r.parents[r.query.Mrn])
}

// Note: you only want to run this, after you are sure that all connected
// dependencies have been processed. Properties must be compiled. Connected
// queries may not be ready yet, but we have to have precompiled them.
func (c *bundleCache) compileQuery(query *explorer.Mquery) {
	props, err := newQueryPropsResolver(query, c.parents)
	if err != nil {
		c.errors = append(c.errors, errors.New("failed to prepare property resolver for query "+query.Mrn))
		return
	}

	_, err = query.RefreshChecksumAndType(c.lookupQuery, props, c.conf.CompilerConfig)
	if err != nil {
		if c.conf.RemoveFailing {
			log.Warn().Err(err).Str("uid", query.Uid).Msg("failed to compile")
			c.removeQueries[query.Mrn] = struct{}{}
		} else {
			c.errors = append(c.errors, multierr.Wrap(err, "failed to validate query '"+query.Mrn+"'"))
		}
	}
	if len(props.errors) != 0 {
		c.errors = append(c.errors, props.errors...)
	}
}

func (c *bundleCache) compileRisk(risk *RiskFactor, policy *Policy) error {
	if err := risk.Filters.Compile(c.ownerMrn, c.conf.CompilerConfig); err != nil {
		c.errors = append(c.errors, errors.New("failed to compile filters for risk factor "+risk.Mrn))
		return nil
	}

	if risk.Filters != nil {
		for j := range risk.Filters.Items {
			filter := risk.Filters.Items[j]
			policy.ComputedFilters.Items[filter.CodeId] = filter
		}
	}

	for i := range risk.Checks {
		check := risk.Checks[i]

		// filters have no dependencies, so we can compile them early
		if err := check.Filters.Compile(c.ownerMrn, c.conf.CompilerConfig); err != nil {
			c.errors = append(c.errors, errors.New("failed to compile filters for risk check "+check.Mrn))
			return nil
		}
		if check.Filters != nil {
			for j := range check.Filters.Items {
				filter := check.Filters.Items[j]
				policy.ComputedFilters.Items[filter.CodeId] = filter
			}
		}

		_, err := check.RefreshChecksumAndType(c.lookupQuery, mqlc.EmptyPropsHandler, c.conf.CompilerConfig)
		if err != nil {
			if c.conf.RemoveFailing {
				panic("REMOVE FAILING risk factors")
				// c.removeQueries[check.Mrn] = struct{}{}
			} else {
				c.errors = append(c.errors, multierr.Wrap(err, "failed to validate query '"+check.Mrn+"'"))
			}
		}
	}

	return nil
}

// for a given policy, translate all local UIDs into global IDs/MRNs
func translateGroupUIDs(ownerMrn string, policyObj *Policy, uid2mrn map[string]string) error {
	for i := range policyObj.Groups {
		group := policyObj.Groups[i]

		for i := range group.Queries {
			query := group.Queries[i]
			if mrn, ok := uid2mrn[query.Uid]; ok {
				query.Mrn = mrn
				query.Uid = ""
			}
		}

		for i := range group.Checks {
			check := group.Checks[i]
			if mrn, ok := uid2mrn[check.Uid]; ok {
				check.Mrn = mrn
				check.Uid = ""
			}
		}

		for i := range group.Policies {
			policy := group.Policies[i]
			if mrn, ok := uid2mrn[policy.Uid]; ok {
				policy.Mrn = mrn
				policy.Uid = ""
			}
		}

	}

	return nil
}

// Takes a query pack bundle and converts it to a policy bundle.
// It copies over the owner, the packs, the props and the queries from the bundle
// and converts all query packs into data-only policies.
func FromQueryPackBundle(bundle *explorer.Bundle) *Bundle {
	if bundle == nil {
		return nil
	}
	b := &Bundle{
		OwnerMrn: bundle.OwnerMrn,
		Packs:    bundle.Packs,
		Props:    bundle.Props,
		Queries:  bundle.Queries,
	}
	b.ConvertQuerypacks()

	return b
}
