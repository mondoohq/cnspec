package policy

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/checksums"
	"go.mondoo.com/cnquery/mrn"
	"go.mondoo.com/cnquery/sortx"
)

type ResolvedFramework struct {
	Mrn                  string
	GraphContentChecksum string
	// ReportTargets tracks which checks/policies/controls report into which controls
	// and frameworks.
	// E.g. ReportTarget[check123] = [controlA, controlB]
	// E.g. ReportTarget[controlA] = [frameworkX]
	ReportTargets map[string][]string
	// ReportSources tracks all the sources that a control or framework pulls
	// data from, i.e. all the checks/policies/controls that provide its data.
	// E.g. ReportSources[controlA] = [check123, check45]
	// E.g. ReportSources[frameworkX] = [controlA, ...]
	ReportSources map[string][]string
}

// Compile takes a framework and prepares it to be stored and further
// used in the backend. It separates the framework definition from the
// framework maps.
func (f *Framework) compile(ctx context.Context, ownerMrn string, cache *bundleCache, library Library) error {
	// 1. we start by turning frameworks and controls from UIDs to MRNs.
	// First, we need all MRNs for existing controls.
	if err := f.refreshMRN(ownerMrn, cache); err != nil {
		return err
	}

	for i := range f.Groups {
		group := f.Groups[i]
		for j := range group.Controls {
			control := group.Controls[j]
			if err := control.refreshMRN(ownerMrn, cache); err != nil {
				return err
			}
		}
	}

	// 2. Now we pass through all the framework maps and update their MRNs,
	// in case they were provided
	for i := range f.FrameworkMaps {
		if err := f.FrameworkMaps[i].compile(ctx, ownerMrn, cache, library); err != nil {
			return err
		}
	}

	return nil
}

func (fm *FrameworkMap) compile(ctx context.Context, ownerMrn string, cache *bundleCache, library Library) error {
	var ok bool

	if err := fm.refreshMRN(ownerMrn, cache); err != nil {
		return err
	}

	if nu, ok := cache.uid2mrn[fm.FrameworkOwner]; ok {
		fm.FrameworkOwner = nu
	}

	for i := range fm.FrameworkDependencies {
		ref := fm.FrameworkDependencies[i]
		if ref.Uid == "" {
			continue
		}
		ref.Mrn, ok = cache.uid2mrn[ref.Uid]
		if !ok {
			return errors.New("cannot find framework dependency '" + ref.Uid + "' in this bundle, which is referenced by framework map " + fm.Mrn)
		}
		ref.Uid = ""
	}

	for i := range fm.PolicyDependencies {
		ref := fm.PolicyDependencies[i]
		if ref.Uid == "" {
			continue
		}
		ref.Mrn, ok = cache.uid2mrn[ref.Uid]
		if !ok {
			return errors.New("cannot find policy dependency '" + ref.Uid + "' in this bundle, which is referenced by framework map " + fm.Mrn)
		}
		ref.Uid = ""
	}

	for j := range fm.Controls {
		control := fm.Controls[j]
		if err := control.refreshMRNs(ownerMrn, cache); err != nil {
			return err
		}
	}
	return nil
}

func checksumControlRef(cr *ControlRef) string {
	c := checksums.
		New.
		Add(cr.Mrn).
		AddUint(uint64(cr.Action))

	return c.String()
}

func (m *FrameworkMap) UpdateChecksums() {
	executionChecksum := checksums.
		New.
		Add(m.Mrn).
		Add(m.FrameworkOwner)

	for _, dep := range m.FrameworkDependencies {
		executionChecksum = executionChecksum.Add(dep.Mrn)
	}

	for _, dep := range m.PolicyDependencies {
		executionChecksum = executionChecksum.Add(dep.Mrn)
	}

	for _, controlMap := range m.Controls {
		executionChecksum = executionChecksum.Add(controlMap.Mrn)
		for _, cr := range controlMap.Checks {
			executionChecksum = executionChecksum.Add(checksumControlRef(cr))
		}
		for _, cr := range controlMap.Policies {
			executionChecksum = executionChecksum.Add(checksumControlRef(cr))
		}
		for _, cr := range controlMap.Controls {
			executionChecksum = executionChecksum.Add(checksumControlRef(cr))
		}
	}

	contentChecksum := checksums.New.
		Add(m.Mrn).
		Add(m.FrameworkOwner).
		Add(executionChecksum.String())

	m.LocalExecutionChecksum = executionChecksum.String()
	m.LocalContentChecksum = contentChecksum.String()
}

func getFrameworkNoop(ctx context.Context, mrn string) (*Framework, error) {
	return nil, errors.New("framework not found: " + mrn)
}

func getFrameworkMapsNoop(ctx context.Context, mrn string) ([]*FrameworkMap, error) {
	return []*FrameworkMap{}, nil
}

func (f *Framework) ClearGraphChecksums() {
	f.GraphContentChecksum = ""
	f.GraphExecutionChecksum = ""
}

func (f *Framework) ClearLocalChecksums() {
	f.LocalContentChecksum = ""
	f.LocalExecutionChecksum = ""
}

func (f *Framework) ClearExecutionChecksums() {
	f.LocalExecutionChecksum = ""
	f.GraphExecutionChecksum = ""
}

func (f *Framework) ClearAllChecksums() {
	f.LocalContentChecksum = ""
	f.LocalExecutionChecksum = ""
	f.GraphContentChecksum = ""
	f.GraphExecutionChecksum = ""
}

func (f *Framework) UpdateChecksums(ctx context.Context,
	getFramework func(ctx context.Context, mrn string) (*Framework, error),
	getFrameworkMaps func(ctx context.Context, mrn string) ([]*FrameworkMap, error),
	bundle *PolicyBundleMap,
) error {
	// simplify the access if we don't have a bundle
	if bundle == nil {
		bundle = &PolicyBundleMap{
			Frameworks: map[string]*Framework{},
		}
	}

	if getFramework == nil {
		getFramework = getFrameworkNoop
	}

	if getFrameworkMaps == nil {
		getFrameworkMaps = getFrameworkMapsNoop
	}

	// if we have local checksums set, we can take an optimized route;
	// if not, we have to update all checksums
	if f.LocalContentChecksum == "" || f.LocalExecutionChecksum == "" {
		return f.updateAllChecksums(ctx, getFramework, getFrameworkMaps, bundle)
	}

	// otherwise we have local checksums and only need to recompute the
	// graph checksums. This code is identical to the complete computation
	// but doesn't recompute any of the local checksums.
	return f.updateGraphChecksums(ctx, getFramework, getFrameworkMaps, bundle)
}

func (f *Framework) updateGraphChecksums(
	ctx context.Context,
	getFramework func(ctx context.Context, mrn string) (*Framework, error),
	getFrameworkMaps func(ctx context.Context, mrn string) ([]*FrameworkMap, error),
	bundle *PolicyBundleMap,
) error {
	graphExecutionChecksum := checksums.New
	graphContentChecksum := checksums.New

	sort.Slice(f.Dependencies, func(i, j int) bool {
		return f.Dependencies[i].Mrn < f.Dependencies[j].Mrn
	})

	for i := range f.Dependencies {
		dep := f.Dependencies[i]

		depObj, ok := bundle.Frameworks[dep.Mrn]
		if !ok {
			var err error
			depObj, err = getFramework(ctx, dep.Mrn)
			if err != nil {
				return err
			}
			frameworkMaps, err := getFrameworkMaps(ctx, dep.Mrn)
			if err != nil {
				return err
			}
			depObj.FrameworkMaps = frameworkMaps
		}

		if err := depObj.UpdateChecksums(ctx, getFramework, getFrameworkMaps, bundle); err != nil {
			return err
		}

		graphExecutionChecksum = graphExecutionChecksum.
			Add(depObj.GraphExecutionChecksum)
		graphContentChecksum = graphContentChecksum.Add(depObj.GraphContentChecksum)
	}

	for _, fm := range f.FrameworkMaps {
		if fm.LocalContentChecksum == "" || fm.LocalExecutionChecksum == "" {
			fm.UpdateChecksums()
		}
		graphExecutionChecksum = graphExecutionChecksum.Add(fm.LocalExecutionChecksum)
		graphContentChecksum = graphContentChecksum.Add(fm.LocalContentChecksum)
	}

	f.GraphExecutionChecksum = graphExecutionChecksum.Add(f.LocalExecutionChecksum).String()
	f.GraphContentChecksum = graphContentChecksum.Add(f.LocalContentChecksum).String()
	return nil
}

func (f *Framework) updateAllChecksums(ctx context.Context,
	getFramework func(ctx context.Context, mrn string) (*Framework, error),
	getFrameworkMaps func(ctx context.Context, mrn string) ([]*FrameworkMap, error),
	bundle *PolicyBundleMap,
) error {
	log.Trace().Str("framework", f.Mrn).Msg("update framework checksum")
	f.LocalContentChecksum = ""
	f.LocalExecutionChecksum = ""

	// Note: this relies on the fact that the bundle was compiled before

	executionChecksum := checksums.New
	contentChecksum := checksums.New

	// content fields in the policy
	contentChecksum = contentChecksum.Add(f.Mrn).Add(f.Name).Add(f.Version).Add(f.OwnerMrn)
	for i := range f.Authors {
		author := f.Authors[i]
		contentChecksum = contentChecksum.Add(author.Email).Add(author.Name)
	}
	contentChecksum = contentChecksum.AddUint(uint64(f.Created)).AddUint(uint64(f.Modified))

	if f.Docs != nil {
		contentChecksum = contentChecksum.Add(f.Docs.Desc)
	}

	// Special handling for asset MRNs: While for most frameworks the MRN is
	// important, for assets that's not the case. We can safely ignore it for
	// the sake of the execution checksum. This also helps to indicate where
	// frameworks overlap.
	if x, _ := mrn.GetResource(f.Mrn, MRN_RESOURCE_ASSET); x != "" {
		executionChecksum = executionChecksum.Add("root")
	} else {
		executionChecksum = executionChecksum.Add(f.Mrn)
	}

	// tags
	keys := sortx.Keys[string](f.Tags)
	for _, k := range keys {
		contentChecksum = contentChecksum.Add(k).Add(f.Tags[k])
	}

	// GROUPS
	for i := range f.Groups {
		group := f.Groups[i]

		sort.Slice(group.Controls, func(i, j int) bool {
			return group.Controls[i].Mrn < group.Controls[j].Mrn
		})

		for j := range group.Controls {
			ctrl := group.Controls[j]
			e, c := ctrl.updateChecksum()
			executionChecksum = executionChecksum.Add(e)
			contentChecksum = contentChecksum.Add(c)
		}

		executionChecksum = executionChecksum.AddUint(uint64(group.Type))

		if group.Docs != nil {
			contentChecksum = contentChecksum.
				Add(group.Docs.Desc).
				Add(group.Docs.Justification)
		}

		// other content fields
		contentChecksum = contentChecksum.
			AddUint(uint64(group.Created)).
			AddUint(uint64(group.Modified)).
			Add(group.Uid).
			Add(group.Title)
		if group.Docs != nil {
			contentChecksum = contentChecksum.
				Add(group.Docs.Desc)
			contentChecksum = contentChecksum.Add(group.Docs.Justification)
		}
	}

	f.LocalExecutionChecksum = executionChecksum.String()
	f.LocalContentChecksum = executionChecksum.AddUint(uint64(contentChecksum)).String()

	return f.updateGraphChecksums(ctx, getFramework, getFrameworkMaps, bundle)
}

func (c *Control) updateChecksum() (string, string) {
	executionChecksum := checksums.New.Add(c.Mrn).AddUint(uint64(c.Action))
	contentChecksum := checksums.New.Add(c.Title)

	keys := sortx.Keys[string](c.Tags)
	for _, key := range keys {
		contentChecksum = contentChecksum.Add(key).Add(c.Tags[key])
	}

	if c.Docs != nil {
		contentChecksum = contentChecksum.
			Add(c.Docs.Desc)
		for i := range c.Docs.Refs {
			ref := c.Docs.Refs[i]
			contentChecksum = contentChecksum.Add(ref.Title).Add(ref.Url)
		}
	}

	if c.Manual {
		executionChecksum = executionChecksum.AddUint(1)
	} else {
		executionChecksum = executionChecksum.AddUint(0)
	}

	contentChecksum = contentChecksum.AddUint(uint64(executionChecksum))
	return executionChecksum.String(), contentChecksum.String()
}

func (f *Framework) refreshMRN(ownerMRN string, cache *bundleCache) error {
	nu, err := RefreshMRN(ownerMRN, f.Mrn, MRN_RESOURCE_FRAMEWORK, f.Uid)
	if err != nil {
		log.Error().Err(err).Str("owner", ownerMRN).Str("uid", f.Uid).Msg("failed to refresh framework mrn")
		return errors.Wrap(err, "failed to refresh mrn for framework "+f.Name)
	}

	if f.Uid != "" {
		cache.uid2mrn[f.Uid] = nu
	}
	f.Mrn = nu
	f.Uid = ""
	return nil
}

func (f *FrameworkMap) refreshMRN(ownerMRN string, cache *bundleCache) error {
	nu, err := RefreshMRN(ownerMRN, f.Mrn, MRN_RESOURCE_FRAMEWORKMAP, f.Uid)
	if err != nil {
		log.Error().Err(err).Str("owner", ownerMRN).Str("uid", f.Uid).Msg("failed to refresh framework mrn")
		return errors.Wrap(err, "failed to refresh mrn for framework map "+f.Uid)
	}

	if f.Uid != "" {
		cache.uid2mrn[f.Uid] = nu
	}
	f.Mrn = nu
	f.Uid = ""
	return nil
}

// refreshMRNs computes a MRN from the UID or validates the existing MRN.
// Both of these need to fit the ownerMRN. It also removes the UID.
func (c *Control) refreshMRN(ownerMRN string, cache *bundleCache) error {
	nu, err := RefreshMRN(ownerMRN, c.Mrn, MRN_RESOURCE_CONTROL, c.Uid)
	if err != nil {
		log.Error().Err(err).Str("owner", ownerMRN).Str("uid", c.Uid).Msg("failed to refresh control mrn")
		return errors.Wrap(err, "failed to refresh mrn for control "+c.Title)
	}

	if c.Uid != "" {
		cache.uid2mrn[c.Uid] = nu
	}
	c.Mrn = nu
	c.Uid = ""
	return nil
}

func (c *ControlMap) refreshMRNs(ownerMRN string, cache *bundleCache) error {
	nu, err := RefreshMRN(ownerMRN, c.Mrn, MRN_RESOURCE_CONTROL, c.Uid)
	if err != nil {
		log.Error().Err(err).Str("owner", ownerMRN).Str("uid", c.Uid).Msg("failed to refresh control mrn")
		return errors.Wrap(err, "failed to refresh mrn for control "+c.Uid)
	}

	if c.Uid != "" {
		cache.uid2mrn[c.Uid] = nu
	}
	c.Mrn = nu
	c.Uid = ""

	var ok bool
	for i := range c.Checks {
		check := c.Checks[i]
		if check.Uid == "" {
			continue
		}
		check.Mrn, ok = cache.uid2mrn[check.Uid]
		if !ok {
			return errors.New("cannot find check '" + check.Uid + "' in this bundle, which is referenced by control " + c.Mrn)
		}
		check.Uid = ""
	}

	for i := range c.Policies {
		policy := c.Policies[i]
		if policy.Uid == "" {
			continue
		}
		policy.Mrn, ok = cache.uid2mrn[policy.Uid]
		if !ok {
			return errors.New("cannot find policy '" + policy.Uid + "' in this bundle, which is referenced by control " + c.Mrn)
		}
		policy.Uid = ""
	}

	for i := range c.Controls {
		control := c.Controls[i]
		if control.Uid == "" {
			continue
		}
		control.Mrn, ok = cache.uid2mrn[control.Uid]
		if !ok {
			return errors.New("cannot find policy '" + control.Uid + "' in this bundle, which is referenced by control " + c.Mrn)
		}
	}

	return nil
}

func ResolveFramework(mrn string, frameworks map[string]*Framework) *ResolvedFramework {
	res := &ResolvedFramework{
		Mrn:           mrn,
		ReportTargets: map[string][]string{},
		ReportSources: map[string][]string{},
	}

	for _, framework := range frameworks {
		for i := range framework.FrameworkMaps {
			fmap := framework.FrameworkMaps[i]

			for _, ctl := range fmap.Controls {
				res.addReportLink(framework.Mrn, ctl.Mrn)
				res.addControl(ctl)
			}
		}
		// FIXME: why do these not show up in the framework map
		for _, depFramework := range framework.Dependencies {
			res.addReportLink(framework.Mrn, depFramework.Mrn)
		}
	}

	return res
}

func (r *ResolvedFramework) addControl(control *ControlMap) {
	for i := range control.Checks {
		r.addReportLink(control.Mrn, control.Checks[i].Mrn)
	}
	for i := range control.Policies {
		r.addReportLink(control.Mrn, control.Policies[i].Mrn)
	}
	for i := range control.Controls {
		r.addReportLink(control.Mrn, control.Controls[i].Mrn)
	}
}

func (r *ResolvedFramework) addReportLink(parent, child string) {
	existing, ok := r.ReportTargets[child]
	if !ok {
		r.ReportTargets[child] = []string{parent}
	} else {
		r.ReportTargets[child] = append(existing, parent)
	}

	existing, ok = r.ReportSources[parent]
	if !ok {
		r.ReportSources[parent] = []string{child}
	} else {
		r.ReportSources[parent] = append(existing, child)
	}
}
