// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package-level entry points for the OCSF reporter: what to convert, the
// options that shape it, and the per-asset walk every event class hangs off.
//
// The mapping of each class lives beside it: ocsf_compliance.go,
// ocsf_detection.go, ocsf_vulnerability.go and ocsf_asset.go, with the
// severity, status and time rules they share in ocsf_mapping.go.

package reporter

import (
	"io"
	"time"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnspec"
	"go.mondoo.com/cnspec/cli/reporter/ocsf"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/upstream/mvd"
)

// The OCSF report carries the same content as the SARIF report, re-cut along the
// event classes a security data lake indexes on:
//
//	Compliance Finding (2003)     one per check per asset, plus one per asset
//	                              that failed to scan at all
//	Vulnerability Finding (2002)  one per advisory per asset, plus one per
//	                              affected package no advisory accounts for
//	Device Inventory Info (5001)  one per asset, carrying the platform details
//	                              and (with the "data" option) the results of
//	                              data-only queries
//
// Every event is self-describing: `class_uid` says what it is, `metadata.version`
// says which OCSF version it follows. See cli/reporter/ocsf for the schema.

// ocsfProductName and friends identify cnspec as the producer of the events.
const (
	ocsfProductName   = "cnspec"
	ocsfProductVendor = "Mondoo, Inc."
	ocsfProductURL    = "https://cnspec.io"
)

// ConvertToOCSF converts a report collection into OCSF events.
func ConvertToOCSF(r *policy.ReportCollection, conf *PrintConfig) (*ocsf.Events, error) {
	return convertToOCSF(r, conf.ocsfConfig(), time.Now())
}

// ConvertToOCSFJSON writes a report collection as newline-delimited OCSF events.
func ConvertToOCSFJSON(r *policy.ReportCollection, conf *PrintConfig, out io.Writer) error {
	return StreamOCSF(r, conf, ocsf.NewJSONWriter(out))
}

// StreamOCSF converts a report collection and writes it out as it goes, so that
// no more than one asset's events are held at a time.
func StreamOCSF(r *policy.ReportCollection, conf *PrintConfig, w ocsf.Writer) error {
	c := newOcsfConverter(conf.ocsfConfig(), time.Now())
	if err := c.stream(r, w.Write); err != nil {
		// Close anyway: a Parquet file without its footer is unreadable, and
		// leaving a half-written one behind is worse than an empty one.
		_ = w.Close()
		return err
	}
	return w.Close()
}

// VulnReportToOCSFJSON writes a standalone vulnerability report (cnspec vuln) as
// newline-delimited OCSF Vulnerability Findings.
func VulnReportToOCSFJSON(target string, data *mvd.VulnReport, version ocsf.Version, out io.Writer) error {
	c := &ocsfConverter{version: version, now: time.Now().UnixMilli()}
	asset := &inventory.Asset{Name: target}
	events := &ocsf.Events{}
	c.addVulnerabilityFindings(events, data, &ocsfAssetContext{
		asset:    asset,
		device:   buildOcsfDevice(asset, version),
		resource: buildOcsfResource(asset),
	})
	events.Sort()
	return events.WriteJSON(out)
}

func convertToOCSF(r *policy.ReportCollection, conf ocsfConfig, now time.Time) (*ocsf.Events, error) {
	return newOcsfConverter(conf, now).convert(r)
}

func newOcsfConverter(conf ocsfConfig, now time.Time) *ocsfConverter {
	return &ocsfConverter{
		version:     conf.version,
		findings:    conf.findings,
		includeData: conf.includeData,
		now:         now.UnixMilli(),
	}
}

// ocsfConfig is what the output format options say about the events to produce.
type ocsfConfig struct {
	version     ocsf.Version
	findings    OcsfFindingClasses
	includeData bool
}

// OcsfFindingClasses selects which OCSF class a check result is reported as.
//
// Every check is reported exactly once, in one class. That is what other OCSF
// producers do: Security Lake routes each Security Hub finding to the one class
// that fits it, and Prowler reports every check as a Detection Finding. Emitting
// a check as two events would double it in anything that counts findings across
// classes.
//
// A cnspec check is a compliance check, so Compliance Finding (2003) is the
// default: it has a compliance object for the framework mappings and the control.
// Detection Finding (2004) is what Splunk Enterprise Security and similar tools
// model findings on; it has no compliance object, so the mappings travel in
// unmapped, and it has the risk and impact attributes 2003 lacks.
type OcsfFindingClasses byte

const (
	OcsfFindingsCompliance OcsfFindingClasses = iota + 1
	OcsfFindingsDetection
)

func (f OcsfFindingClasses) compliance() bool { return f == OcsfFindingsCompliance }

func (f OcsfFindingClasses) detection() bool { return f == OcsfFindingsDetection }

// ocsfConverter holds everything that is the same for every event of one run.
type ocsfConverter struct {
	version     ocsf.Version
	findings    OcsfFindingClasses
	includeData bool
	// now is the scan time in milliseconds since the epoch. It is a field rather
	// than a call to time.Now so that a conversion is reproducible.
	now int64
}

// ocsfAssetContext carries the per-asset values every event of that asset needs.
type ocsfAssetContext struct {
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
func (c *ocsfConverter) stream(r *policy.ReportCollection, emit func(*ocsf.Events) error) error {
	if r == nil {
		return nil
	}
	if r.Bundle == nil {
		return errors.New("no policy bundle found")
	}

	bundle := r.Bundle.ToMap()
	queries := reporterQueryMap(bundle)
	policyTitles := policyTitlesByQuery(bundle)

	for _, assetMrn := range sortedKeys(r.Assets) {
		asset := r.Assets[assetMrn]
		if asset == nil {
			continue
		}
		ctx := &ocsfAssetContext{
			assetMrn:     assetMrn,
			asset:        asset,
			platformKeys: platformRemediationKeys(asset.Platform),
			policyTitles: policyTitles,
			device:       buildOcsfDevice(asset, c.version),
			cloud:        buildOcsfCloud(asset),
			resource:     buildOcsfResource(asset),
		}

		events := &ocsf.Events{}
		events.InventoryInfos = append(events.InventoryInfos, c.inventoryInfo(r, ctx))
		c.addComplianceFindings(events, r, ctx, queries)
		c.addAssetError(events, r, ctx)
		c.addVulnerabilityFindings(events, r.VulnReports[assetMrn], ctx)
		events.Sort()

		if err := emit(events); err != nil {
			return err
		}
	}
	return nil
}

// convert accumulates the whole scan in memory. Callers that write the events
// out should stream instead; this is for the ones that need the complete set,
// such as tests and the standalone vulnerability report.
func (c *ocsfConverter) convert(r *policy.ReportCollection) (*ocsf.Events, error) {
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
func (c *ocsfConverter) metadata(profiles ...string) ocsf.Metadata {
	res := ocsf.Metadata{
		Version: string(c.version),
		Product: ocsf.Product{
			Name:       ocsfProductName,
			VendorName: ocsfProductVendor,
			Version:    cnspec.GetVersion(),
			URLString:  ocsfProductURL,
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
func (ctx *ocsfAssetContext) findingProfiles() []string {
	res := []string{}
	if ctx.device != nil {
		res = append(res, ocsf.ProfileHost)
	}
	if ctx.cloud != nil {
		res = append(res, ocsf.ProfileCloud)
	}
	return res
}
