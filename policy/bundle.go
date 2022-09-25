package policy

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/checksums"
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
