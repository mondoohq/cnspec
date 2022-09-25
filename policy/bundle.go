package policy

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/checksums"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/mrn"
	"sigs.k8s.io/yaml"
)

// BundleFromPaths loads a single policy bundle file or a bundle that
// was split into multiple files into a single PolicyBundle struct
func BundleFromPaths(paths ...string) (*PolicyBundle, error) {
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
				// we ignore nested directories
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
func aggregateFilesToBundle(paths []string) (*PolicyBundle, error) {
	// iterate over all files, load them and merge them
	mergedBundle := &PolicyBundle{}

	for i := range paths {
		path := paths[i]
		bundle, err := bundleFromSingleFile(path)
		if err != nil {
			return nil, errors.Wrap(err, "could not load file: "+path)
		}

		mergedBundle = aggregateBundles(mergedBundle, bundle)
	}

	return mergedBundle, nil
}

// bundleFromSingleFile loads a policy bundle from a single file
func bundleFromSingleFile(path string) (*PolicyBundle, error) {
	bundleData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return BundleFromYAML(bundleData)
}

// aggregateBundles combines two PolicyBundle and merges the data additive into one
// single PolicyBundle structure
func aggregateBundles(a *PolicyBundle, b *PolicyBundle) *PolicyBundle {
	res := &PolicyBundle{}

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
func BundleFromYAML(data []byte) (*PolicyBundle, error) {
	var res PolicyBundle
	err := yaml.Unmarshal(data, &res)
	return &res, err
}

// ToYAML returns the policy bundle as yaml
func (p *PolicyBundle) ToYAML() ([]byte, error) {
	return yaml.Marshal(p)
}

func (p *PolicyBundle) SourceHash() (string, error) {
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
func (p *PolicyBundle) ToMap() *PolicyBundleMap {
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

// Clean the policy bundle to turn a few nil fields into empty fields for consistency
func (p *PolicyBundle) Clean() *PolicyBundle {
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

// Compile PolicyBundle into a PolicyBundleMap
// Does 4 things:
// 1. turns policy bundle into a map for easier access
// 2. compile all queries. store code in the bundle map
// 3. validation of all contents
// 4. generate MRNs for all policies, queries, and properties and updates referencing local fields
func (p *PolicyBundle) Compile(ctx context.Context, library Library) (*PolicyBundleMap, error) {
	ownerMrn := p.OwnerMrn
	if ownerMrn == "" {
		return nil, errors.New("failed to compile bundle, the owner MRN is empty")
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

		err = query.RefreshMrn(ownerMrn)
		if err != nil {
			return nil, errors.New("failed to refresh property: " + err.Error())
		}

		// recalculate the checksums
		bundle, err := query.RefreshChecksumAndType(props)
		if err != nil {
			return nil, errors.New("failed to validate property '" + query.Mrn + "': " + err.Error())
		}

		name, err := mrn.GetResource(query.Mrn, "query")
		if err != nil {
			return nil, errors.New("could not read property name from query mrn: " + query.Mrn)
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

		err := policy.RefreshMrn(ownerMrn)
		if err != nil {
			return nil, errors.New("failed to refresh policy " + policy.Mrn + ": " + err.Error())
		}

		if policyUID != "" {
			uid2mrn[policyUID] = policy.Mrn
		}

		// Properties
		for name, target := range policy.Props {
			if target != "" {
				return nil, errors.New("overwriting properties not yet supported - sorryyyy")
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

		// remove leading and trailing whitespace of docs, refs and tags
		query.Sanitize()

		// ensure the correct mrn is set
		uid := query.Uid
		if err = query.RefreshMrn(ownerMrn); err != nil {
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
