// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package convert

import (
	"io"
	"maps"
	"slices"
	"time"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/cnspec/reports/ocsf"
	"go.mondoo.com/cnspec/reports/reportdoc"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/mvd"
)

// productName and friends identify cnspec as the producer of the events.
const (
	productName   = "cnspec"
	productVendor = "Mondoo, Inc."
	productURL    = "https://cnspec.io"
)

// Options is what the caller says about the events to produce. The zero value is
// usable: it produces the default version and the default finding class, because
// an unset Version would otherwise be written into metadata.version as an empty
// string and pass no validator anywhere.
type Options struct {
	// Version is the OCSF schema version the events follow.
	Version ocsf.Version
	// Findings selects which OCSF class check results are reported as.
	Findings ocsf.FindingClasses
	// IncludeData carries the results of the scan's data-only queries on the
	// inventory event.
	IncludeData bool
}

func (o Options) withDefaults() Options {
	if o.Version == "" {
		o.Version = ocsf.DefaultVersion
	}
	if o.Findings == 0 {
		o.Findings = ocsf.FindingsCompliance
	}
	return o
}

// Convert writes a scan as one newline-delimited stream of OCSF events, every
// class mixed together in one document. That is what a SIEM ingesting a single
// file expects; a data lake wants a file per class, which is ConvertToDir.
func Convert(r *policy.ReportCollection, out io.Writer, opts Options) error {
	return Stream(r, ocsf.NewJSONWriter(out), opts)
}

// Stream converts a scan and hands it to a writer as it goes, so that no more
// than one asset's events are held at a time.
func Stream(r *policy.ReportCollection, w ocsf.Writer, opts Options) error {
	if err := newConverter(opts, time.Now()).stream(r, w.Write); err != nil {
		// Close anyway: a Parquet file without its footer is unreadable, and
		// leaving a half-written one behind is worse than an empty one.
		_ = w.Close()
		return err
	}
	return w.Close()
}

// ConvertVulnReport writes a standalone vulnerability report (cnspec vuln) as
// newline-delimited OCSF Vulnerability Findings. It has no scan behind it, so
// the target names the asset and there is nothing else to report about it.
func ConvertVulnReport(target string, data *mvd.VulnReport, version ocsf.Version, out io.Writer) error {
	c := newConverter(Options{Version: version}, time.Now())
	asset := &inventory.Asset{Name: target}
	events := &ocsf.Events{}
	c.addVulnerabilityFindings(events, data, &assetContext{
		asset:    asset,
		device:   buildDevice(asset, c.version),
		resource: buildResource(asset),
	})
	events.Sort()
	return events.WriteJSON(out)
}

// convertAt accumulates the whole scan in memory at a fixed conversion time.
// Callers that write the events out should stream instead; this is for the ones
// that need the complete set, which in practice is the tests.
func convertAt(r *policy.ReportCollection, opts Options, now time.Time) (*ocsf.Events, error) {
	return newConverter(opts, now).convert(r)
}

func newConverter(opts Options, now time.Time) *converter {
	opts = opts.withDefaults()
	return &converter{
		version:     opts.Version,
		findings:    opts.Findings,
		includeData: opts.IncludeData,
		now:         now.UnixMilli(),
	}
}

// converter holds everything that is the same for every event of one run.
type converter struct {
	version     ocsf.Version
	findings    ocsf.FindingClasses
	includeData bool
	// now is the scan time in milliseconds since the epoch. It is a field rather
	// than a call to time.Now so that a conversion is reproducible.
	now int64
}

// detection reports whether checks go out as class 2004 rather than 2003.
func (c *converter) detection() bool { return c.findings == ocsf.FindingsDetection }

// assetContext carries the per-asset values every event of that asset needs.
type assetContext struct {
	assetMrn     string
	asset        *inventory.Asset
	platformKeys map[string]bool
	policyTitles map[string][]string
	device       *ocsf.Device
	cloud        *ocsf.Cloud
	resource     ocsf.ResourceDetails
}

// stream converts the scan one asset at a time and hands each asset's events to
// emit.
//
// Converting asset by asset is what keeps memory flat: a writer encodes each set
// and lets it go, so peak usage tracks the largest asset rather than the size of
// the whole report. A fleet scan produces events by the hundred thousand, and
// holding them all costs several KB each.
//
// The walk is ordered — assets by MRN, checks by id — so a streamed run is as
// reproducible as a materialized one, just grouped by asset rather than by
// finding.
func (c *converter) stream(r *policy.ReportCollection, emit func(*ocsf.Events) error) error {
	if r == nil {
		return nil
	}
	if r.Bundle == nil {
		return errors.New("no policy bundle found")
	}

	bundle := r.Bundle.ToMap()
	queries := reportdoc.QueryMap(bundle)
	policyTitles := reportdoc.PolicyTitlesByQuery(bundle)

	for _, assetMrn := range slices.Sorted(maps.Keys(r.Assets)) {
		asset := r.Assets[assetMrn]
		if asset == nil {
			continue
		}
		ctx := &assetContext{
			assetMrn:     assetMrn,
			asset:        asset,
			platformKeys: reportdoc.PlatformRemediationKeys(asset.Platform),
			policyTitles: policyTitles,
			device:       buildDevice(asset, c.version),
			cloud:        buildCloud(asset, c.version),
			resource:     buildResource(asset),
		}

		events := &ocsf.Events{}
		events.InventoryInfos = append(events.InventoryInfos, c.inventoryInfo(r, ctx))
		c.addCheckFindings(events, r, ctx, queries)
		c.addAssetError(events, r, ctx)
		c.addVulnerabilityFindings(events, r.VulnReports[assetMrn], ctx)
		events.Sort()

		if err := emit(events); err != nil {
			return err
		}
	}
	return nil
}

// convert accumulates the whole scan in memory.
func (c *converter) convert(r *policy.ReportCollection) (*ocsf.Events, error) {
	events := &ocsf.Events{}
	err := c.stream(r, func(chunk *ocsf.Events) error {
		events.Append(chunk)
		return nil
	})
	if err != nil {
		return nil, err
	}
	events.Sort()
	return events, nil
}

// metadata is the same for every event of a run, except for the profiles.
//
// Profiles are not decoration: an OCSF attribute that belongs to a profile is
// only allowed on an event that declares the profile. `device` on a finding
// comes from the host profile and `cloud` from the cloud profile, so an event
// carrying either without saying so is rejected by a schema-aware validator.
func (c *converter) metadata(profiles ...string) ocsf.Metadata {
	res := ocsf.Metadata{
		Version: string(c.version),
		Product: ocsf.Product{
			Name:       productName,
			VendorName: productVendor,
			Version:    cnspec.GetVersion(),
			URLString:  productURL,
		},
		LoggedTime: c.now,
	}
	if len(profiles) > 0 {
		res.Profiles = profiles
	}
	return res
}

// findingProfiles lists the OCSF profiles a finding of this asset uses. Findings
// carry the asset as a device (host profile) and, for cloud assets, the cloud
// environment (cloud profile).
func (ctx *assetContext) findingProfiles() []string {
	res := []string{}
	if ctx.device != nil {
		res = append(res, ocsf.ProfileHost)
	}
	if ctx.cloud != nil {
		res = append(res, ocsf.ProfileCloud)
	}
	return res
}
