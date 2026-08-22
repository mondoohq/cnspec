// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Device Inventory Info (OCSF class 5001) and the asset itself: the device, cloud
// and resource objects every event class reuses.

package reporter

import (
	"bytes"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/cli/reporter/ocsf"
	"go.mondoo.com/cnspec/policy"
	cr "go.mondoo.com/mql/cli/reporter"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/utils/iox"
)

// inventoryInfo reports the asset itself, so the lake knows what was scanned even
// when the scan produced no findings.
func (c *ocsfConverter) inventoryInfo(r *policy.ReportCollection, ctx *ocsfAssetContext) ocsf.InventoryInfo {
	res := ocsf.NewInventoryInfo(ocsf.InventoryInfoActivityCollect)
	res.Cloud = ctx.cloud
	if ctx.device != nil {
		res.Device = *ctx.device
	}
	res.Time = c.now
	res.SeverityID = ocsf.SeverityInformational
	res.Severity = ocsf.SeverityName(res.SeverityID)
	// device is a class attribute of inventory_info, not a host-profile one, so
	// only the cloud profile has to be declared here.
	var profiles []string
	if ctx.cloud != nil {
		profiles = append(profiles, ocsf.ProfileCloud)
	}
	res.Metadata = c.metadata(profiles...)
	res.Unmapped = c.assetUnmapped(r, ctx)
	return res
}

// assetUnmapped carries the asset details and, when data output is enabled, the
// results of the data-only queries of the scan.
func (c *ocsfConverter) assetUnmapped(r *policy.ReportCollection, ctx *ocsfAssetContext) map[string]string {
	res := map[string]string{}
	if ctx.assetMrn != "" {
		res["asset_mrn"] = ctx.assetMrn
	}
	asset := ctx.asset
	if len(asset.PlatformIds) > 0 {
		res["platform_ids"] = strings.Join(asset.PlatformIds, ",")
	}
	for k, v := range asset.Labels {
		res["label."+k] = v
	}
	for k, v := range asset.Annotations {
		res["annotation."+k] = v
	}
	if asset.Platform != nil {
		if asset.Platform.Kind != "" {
			res["platform_kind"] = asset.Platform.Kind
		}
		if asset.Platform.Runtime != "" {
			res["platform_runtime"] = asset.Platform.Runtime
		}
	}

	if !c.includeData {
		return res
	}
	for mrn, value := range assetDataResults(r, ctx.assetMrn) {
		res["data."+mrn] = value
	}
	return res
}

// assetDataResults renders the results of the asset's data-only queries as JSON,
// keyed by query MRN.
func assetDataResults(r *policy.ReportCollection, assetMrn string) map[string]string {
	report, ok := r.Reports[assetMrn]
	if !ok {
		return nil
	}
	resolved, ok := r.ResolvedPolicies[assetMrn]
	if !ok || resolved.ExecutionJob == nil || resolved.CollectorJob == nil {
		return nil
	}

	qid2mrn := map[string]string{}
	if r.Bundle != nil {
		for _, query := range r.Bundle.Queries {
			if query.CodeId != "" {
				qid2mrn[query.CodeId] = query.Mrn
			}
		}
	}

	reportingJobs := map[string]*policy.ReportingJob{}
	for _, job := range resolved.CollectorJob.ReportingJobs {
		reportingJobs[job.QrId] = job
	}

	results := report.RawResults()
	res := map[string]string{}
	for qid, query := range resolved.ExecutionJob.Queries {
		mrn := qid2mrn[qid]
		if mrn == "" {
			continue
		}
		if job, ok := reportingJobs[mrn]; ok &&
			job.Type != policy.ReportingJob_DATA_QUERY && job.Type != policy.ReportingJob_CHECK_AND_DATA_QUERY {
			continue
		}

		buf := &bytes.Buffer{}
		w := iox.IOWriter{Writer: buf}
		if err := cr.CodeBundleToJSON(query.Code, results, &w); err != nil {
			log.Warn().Err(err).Str("query", mrn).Msg("could not render a data query result for the OCSF report")
			continue
		}
		res[mrn] = strings.TrimSpace(buf.String())
	}
	return res
}

// buildOcsfResource describes the scanned asset as the resource a finding is about.
func buildOcsfResource(asset *inventory.Asset) ocsf.ResourceDetails {
	res := ocsf.ResourceDetails{
		UID:  asset.Mrn,
		Name: asset.Name,
	}
	if res.UID == "" && len(asset.PlatformIds) > 0 {
		res.UID = asset.PlatformIds[0]
	}
	if asset.Platform != nil {
		res.Type = asset.Platform.Title
		if res.Type == "" {
			res.Type = asset.Platform.Name
		}
		res.Version = asset.Platform.Version
	}
	for _, key := range sortedKeys(asset.Labels) {
		res.Labels = append(res.Labels, key+"="+asset.Labels[key])
	}
	if arn, ok := awsARN(asset); ok {
		res.CloudPartition = arn.partition
		res.Region = arn.region
	}
	return res
}

// buildOcsfDevice describes the scanned asset as an endpoint.
func buildOcsfDevice(asset *inventory.Asset, version ocsf.Version) *ocsf.Device {
	res := &ocsf.Device{
		TypeID:   ocsf.DeviceTypeOther,
		UID:      asset.Mrn,
		Name:     asset.Name,
		Hostname: asset.Fqdn,
	}
	if res.UID == "" && len(asset.PlatformIds) > 0 {
		res.UID = asset.PlatformIds[0]
	}

	platform := asset.Platform
	if platform == nil {
		return res
	}

	switch platform.Kind {
	case inventory.AssetKindBaremetal:
		res.TypeID = ocsf.DeviceTypeServer
	case inventory.AssetKindCloudVM, "virtualmachine-image":
		res.TypeID = ocsf.DeviceTypeVirtual
	}
	res.Type = ocsfDeviceTypeName(res.TypeID, platform)

	if osType := ocsfOsType(platform); osType != ocsf.OSTypeUnknown {
		res.OS = &ocsf.OS{
			Name:    platform.Name,
			TypeID:  osType,
			Type:    ocsf.OSTypeName(osType),
			Version: platform.Version,
			Build:   platform.Build,
		}
	}
	if platform.Arch != "" {
		// 1.9 deprecates cpu_type in favor of a per-processor cpu_info_list.
		if version.AtLeast(ocsf.Version190) {
			res.HwInfo = &ocsf.DeviceHwInfo{
				CPUInfoList: []ocsf.CPUInfo{{CPUArchitecture: platform.Arch}},
			}
		} else {
			res.HwInfo = &ocsf.DeviceHwInfo{CPUType: platform.Arch}
		}
	}
	if arn, ok := awsARN(asset); ok {
		res.Region = arn.region
	}
	return res
}

// ocsfDeviceTypeName is the string sibling of device.type_id. For a device that
// is none of OCSF's known types the sibling carries what cnspec calls the
// platform, which is more useful than the word "Other" and is what OCSF asks for.
func ocsfDeviceTypeName(typeID int, platform *inventory.Platform) string {
	if typeID != ocsf.DeviceTypeOther {
		return ocsf.DeviceTypeName(typeID)
	}
	if platform == nil {
		return ""
	}
	if platform.Title != "" {
		return platform.Title
	}
	return platform.Name
}

// ocsfOsType detects the operating system family of a platform. Assets that are
// not operating systems (cloud APIs, SaaS, Kubernetes objects) have none.
func ocsfOsType(platform *inventory.Platform) int {
	families := append([]string{platform.Name}, platform.Family...)
	for _, family := range families {
		switch strings.ToLower(family) {
		case "windows":
			return ocsf.OSTypeWindows
		case "linux":
			return ocsf.OSTypeLinux
		case "darwin", "macos":
			return ocsf.OSTypeMacOS
		case "android":
			return ocsf.OSTypeAndroid
		case "solaris":
			return ocsf.OSTypeSolaris
		case "aix":
			return ocsf.OSTypeAIX
		case "hpux":
			return ocsf.OSTypeHPUX
		case "unix", "bsd":
			return ocsf.OSTypeOther
		}
	}
	return ocsf.OSTypeUnknown
}

// buildOcsfCloud fills in the cloud environment of an asset. It returns nil for
// assets that are not cloud resources, which keeps the cloud profile off those
// events.
func buildOcsfCloud(asset *inventory.Asset) *ocsf.Cloud {
	provider := ocsfCloudProvider(asset)
	if provider == "" {
		return nil
	}

	res := &ocsf.Cloud{Provider: provider}
	if arn, ok := awsARN(asset); ok {
		res.Region = arn.region
		if arn.account != "" {
			res.Account = &ocsf.Account{UID: arn.account, Type: "AWS Account"}
		}
	}
	for _, id := range asset.PlatformIds {
		if project, ok := platformIDSegment(id, "/runtime/gcp/projects/"); ok {
			res.ProjectUID = project
		}
		if sub, ok := platformIDSegment(id, "/runtime/azure/subscriptions/"); ok {
			res.Account = &ocsf.Account{UID: sub, Type: "Azure Subscription"}
		}
		if account, ok := platformIDSegment(id, "/runtime/aws/accounts/"); ok {
			res.Account = &ocsf.Account{UID: account, Type: "AWS Account"}
		}
	}
	return res
}

// ocsfCloudProvider reports the cloud an asset belongs to, using the same
// vocabulary OCSF consumers expect ("AWS", "Azure", "GCP").
func ocsfCloudProvider(asset *inventory.Asset) string {
	candidates := []string{}
	if asset.Platform != nil {
		candidates = append(candidates, asset.Platform.Runtime, asset.Platform.Name)
	}
	candidates = append(candidates, asset.PlatformIds...)

	for _, candidate := range candidates {
		switch {
		case candidate == "":
			continue
		case strings.HasPrefix(candidate, "arn:aws"), strings.HasPrefix(candidate, "aws"),
			strings.Contains(candidate, "/runtime/aws/"):
			return "AWS"
		case strings.HasPrefix(candidate, "azure"), strings.Contains(candidate, "/runtime/azure/"):
			return "Azure"
		case strings.HasPrefix(candidate, "gcp"), strings.HasPrefix(candidate, "google"),
			strings.Contains(candidate, "/runtime/gcp/"):
			return "GCP"
		}
	}
	return ""
}

type parsedARN struct {
	partition string
	region    string
	account   string
}

// awsARN pulls the partition, region and account out of the first ARN among an
// asset's platform ids. An ARN is
// arn:<partition>:<service>:<region>:<account>:<resource>, and region and account
// are empty for global and account-less resources.
func awsARN(asset *inventory.Asset) (parsedARN, bool) {
	for _, id := range asset.PlatformIds {
		if !strings.HasPrefix(id, "arn:") {
			continue
		}
		parts := strings.SplitN(id, ":", 6)
		if len(parts) < 5 {
			continue
		}
		return parsedARN{partition: parts[1], region: parts[3], account: parts[4]}, true
	}
	return parsedARN{}, false
}

// platformIDSegment returns the segment that follows a marker in a platform id,
// e.g. the project of //platformid.api.mondoo.app/runtime/gcp/projects/my-project.
func platformIDSegment(id, marker string) (string, bool) {
	idx := strings.Index(id, marker)
	if idx < 0 {
		return "", false
	}
	rest := id[idx+len(marker):]
	if end := strings.Index(rest, "/"); end >= 0 {
		rest = rest[:end]
	}
	if rest == "" {
		return "", false
	}
	return rest, true
}
