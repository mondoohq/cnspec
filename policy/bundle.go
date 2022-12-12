package policy

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/checksums"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/mrn"
	"sigs.k8s.io/yaml"
)

const (
	MRN_RESOURCE_QUERY  = "queries"
	MRN_RESOURCE_POLICY = "policies"
	MRN_RESOURCE_ASSET  = "assets"
)

// BundleFromPaths loads a single policy bundle file or a bundle that
// was split into multiple files into a single PolicyBundle struct
func BundleFromPaths(paths ...string) (*Bundle, error) {
	// load all the source files
	resolvedFilenames, err := walkPolicyBundleFiles(paths)
	if err != nil {
		log.Error().Err(err).Msg("could not resolve bundle files")
		return nil, err
	}

	// aggregate all files into a single policy bundle
	aggregatedBundle, err := aggregateFilesToBundle(resolvedFilenames)
	if err != nil {
		log.Error().Err(err).Msg("could merge bundle files")
		return nil, err
	}

	logger.DebugDumpYAML("resolved_mql_bundle.mql", aggregatedBundle)
	return aggregatedBundle, nil
}

// walkPolicyBundleFiles iterates over all provided filenames and
// checks if the name is a file or a directory. If the filename
// is a directory, it walks the directory recursively
func walkPolicyBundleFiles(filenames []string) ([]string, error) {
	// resolve file names
	resolvedFilenames := []string{}
	for i := range filenames {
		filename := filenames[i]
		fi, err := os.Stat(filename)
		if err != nil {
			return nil, errors.Wrap(err, "could not load policy bundle file: "+filename)
		}

		if fi.IsDir() {
			filepath.WalkDir(filename, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				// we ignore directories because WalkDir already walks them
				if d.IsDir() {
					return nil
				}

				// only consider .yaml|.yml files
				if strings.HasSuffix(d.Name(), ".yaml") || strings.HasSuffix(d.Name(), ".yml") {
					resolvedFilenames = append(resolvedFilenames, path)
				}

				return nil
			})
		} else {
			resolvedFilenames = append(resolvedFilenames, filename)
		}
	}

	return resolvedFilenames, nil
}

// aggregateFilesToBundle iterates over all provided files and loads its content.
// It assumes that all provided files are checked upfront and are not a directory
func aggregateFilesToBundle(paths []string) (*Bundle, error) {
	// iterate over all files, load them and merge them
	mergedBundle := &Bundle{}

	for i := range paths {
		path := paths[i]
		log.Debug().Str("path", path).Msg("loading policy bundle file")
		bundle, err := bundleFromSingleFile(path)
		if err != nil {
			return nil, errors.Wrap(err, "could not load file: "+path)
		}

		mergedBundle = aggregateBundles(mergedBundle, bundle)
	}

	return mergedBundle, nil
}

// bundleFromSingleFile loads a policy bundle from a single file
func bundleFromSingleFile(path string) (*Bundle, error) {
	bundleData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return BundleFromYAML(bundleData)
}

// aggregateBundles combines two PolicyBundle and merges the data additive into one
// single PolicyBundle structure
func aggregateBundles(a *Bundle, b *Bundle) *Bundle {
	res := &Bundle{}

	res.OwnerMrn = a.OwnerMrn
	if b.OwnerMrn != "" {
		res.OwnerMrn = b.OwnerMrn
	}

	// merge in a
	for i := range a.Policies {
		p := a.Policies[i]
		res.Policies = append(res.Policies, p)
	}

	for i := range a.Props {
		p := a.Props[i]
		res.Props = append(res.Props, p)
	}

	for i := range a.Queries {
		q := a.Queries[i]
		res.Queries = append(res.Queries, q)
	}

	// merge in b
	for i := range b.Policies {
		p := b.Policies[i]
		res.Policies = append(res.Policies, p)
	}

	for i := range b.Props {
		p := b.Props[i]
		res.Props = append(res.Props, p)
	}

	for i := range b.Queries {
		q := b.Queries[i]
		res.Queries = append(res.Queries, q)
	}

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
		if policy.AssetFilters == nil {
			policy.AssetFilters = map[string]*Mquery{}
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

// SortContents of this policy bundle sorts Queries and Policies by MRNs
func (p *Bundle) SortContents() {
	sort.SliceStable(p.Queries, func(i, j int) bool {
		return p.Queries[i].Mrn < p.Queries[j].Mrn
	})

	sort.SliceStable(p.Policies, func(i, j int) bool {
		return p.Policies[i].Mrn < p.Policies[j].Mrn
	})
}

// Compile PolicyBundle into a PolicyBundleMap
// Does 4 things:
// 1. turns policy bundle into a map for easier access
// 2. compile all queries. store code in the bundle map
// 3. validation of all contents
// 4. generate MRNs for all policies, queries, and properties and updates referencing local fields
func (p *Bundle) Compile(ctx context.Context, library Library) (*PolicyBundleMap, error) {
	ownerMrn := p.OwnerMrn
	if ownerMrn == "" {
		// this only happens for local bundles where queries have no mrn yet
		ownerMrn = "//local.cnspec.io/run/local-execution"
	}

	var err error
	var warnings []error

	uid2mrn := map[string]string{}
	bundles := map[string]*llx.CodeBundle{}

	// Index properties
	propQueries := map[string]*Mquery{}
	props := map[string]*llx.Primitive{}
	for i := range p.Props {
		query := p.Props[i]

		err = query.RefreshMRN(ownerMrn)
		if err != nil {
			return nil, errors.New("failed to refresh property: " + err.Error())
		}

		// recalculate the checksums
		bundle, err := query.RefreshChecksumAndType(props)
		if err != nil {
			return nil, errors.New("failed to validate property '" + query.Mrn + "': " + err.Error())
		}

		name, err := mrn.GetResource(query.Mrn, MRN_RESOURCE_QUERY)
		if err != nil {
			return nil, errors.New("failed to compile, could not read property name from query mrn: " + query.Mrn)
		}
		propQueries[name] = query
		propQueries[query.Mrn] = query
		props[name] = &llx.Primitive{Type: query.Type} // placeholder
		bundles[query.Mrn] = bundle
	}

	// Index policies + update MRNs and checksums, link properties via MRNs
	for i := range p.Policies {
		policy := p.Policies[i]

		// make sure we get a copy of the UID before it is removed (via refresh MRN)
		policyUID := policy.Uid

		// !this is very important to prevent user overrides! vv
		policy.InvalidateAllChecksums()

		err := policy.RefreshMRN(ownerMrn)
		if err != nil {
			return nil, errors.New("failed to refresh policy " + policy.Mrn + ": " + err.Error())
		}

		if policyUID != "" {
			uid2mrn[policyUID] = policy.Mrn
		}

		// Properties
		for name, target := range policy.Props {
			if target != "" {
				return nil, errors.New("overwriting properties not yet supported - sorry")
			}

			q, ok := propQueries[name]
			if !ok {
				return nil, errors.New("cannot find property '" + name + "' in policy '" + policy.Name + "'")
			}

			// turn UID/name references into MRN references
			if name != q.Mrn {
				delete(policy.Props, name)
				policy.Props[q.Mrn] = target
			}
		}
	}

	// Index queries + update MRNs and checksums
	for i := range p.Queries {
		query := p.Queries[i]
		if query == nil {
			return nil, errors.New("received null query")
		}

		// remove leading and trailing whitespace of docs, refs and tags
		query.Sanitize()

		// ensure the correct mrn is set
		uid := query.Uid
		if err = query.RefreshMRN(ownerMrn); err != nil {
			return nil, err
		}
		if uid != "" {
			uid2mrn[uid] = query.Mrn
		}

		// recalculate the checksums
		bundle, err := query.RefreshChecksumAndType(props)
		if err != nil {
			log.Error().Err(err).Msg("could not compile the query")
			warnings = append(warnings, errors.Wrap(err, "failed to validate query '"+query.Mrn+"'"))
		}

		bundles[query.Mrn] = bundle
	}

	// cannot be done before all policies and queries have their MRNs set
	bundleMap := p.ToMap()
	bundleMap.Library = library
	bundleMap.Code = bundles

	// Validate integrity of references + translate UIDs to MRNs
	for i := range p.Policies {
		policy := p.Policies[i]
		if policy == nil {
			return nil, errors.New("received null policy")
		}

		err := translateSpecUIDs(ownerMrn, policy, uid2mrn)
		if err != nil {
			return nil, errors.New("failed to validate policy: " + err.Error())
		}

		err = bundleMap.ValidatePolicy(ctx, policy)
		if err != nil {
			return nil, errors.New("failed to validate policy: " + err.Error())
		}
	}

	if len(warnings) != 0 {
		var msg strings.Builder
		for i := range warnings {
			msg.WriteString(warnings[i].Error())
			msg.WriteString("\n")
		}
		return bundleMap, errors.New(msg.String())
	}

	return bundleMap, nil
}

// for a given policy, translate all local UIDs in all specs into global IDs/MRNs
func translateSpecUIDs(ownerMrn string, policyObj *Policy, uid2mrn map[string]string) error {
	for i := range policyObj.Specs {
		spec := policyObj.Specs[i]

		if len(spec.DataQueries) > 0 {
			datamap := map[string]QueryAction{}
			for k, v := range spec.DataQueries {
				nuID, ok := uid2mrn[k]
				if !ok {
					datamap[k] = v
				} else {
					datamap[nuID] = v
				}
			}
			spec.DataQueries = datamap
		}

		if len(spec.ScoringQueries) > 0 {
			scorings := map[string]*ScoringSpec{}
			for k, v := range spec.ScoringQueries {
				nuID, ok := uid2mrn[k]
				if !ok {
					scorings[k] = v
				} else {
					scorings[nuID] = v
				}
			}
			spec.ScoringQueries = scorings
		}

		if len(spec.Policies) > 0 {
			policies := map[string]*ScoringSpec{}
			for k, v := range spec.Policies {
				nuID, ok := uid2mrn[k]
				if ok {
					policies[nuID] = v
					continue
				}

				if mrn.IsValid(k) {
					policies[k] = v
					continue
				}

				return errors.New("found a policy reference which is neither an MRN nor in this bundle: " + k)
			}
			spec.Policies = policies
		}
	}

	return nil
}
