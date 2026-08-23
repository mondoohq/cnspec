// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// The device, cloud and resource objects every OCSF event class carries. They are
// built once per asset in assetContext and reused by each event of that asset,
// because they cost a platform-id parse each and every class needs them.

package convert

import (
	"maps"
	"slices"
	"strings"

	"go.mondoo.com/cnspec/reports/ocsf"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// buildResource describes the scanned asset as the resource a finding is about.
func buildResource(asset *inventory.Asset) ocsf.ResourceDetails {
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
	for _, key := range slices.Sorted(maps.Keys(asset.Labels)) {
		res.Labels = append(res.Labels, key+"="+asset.Labels[key])
	}
	if arn, ok := awsARN(asset); ok {
		res.CloudPartition = arn.partition
		res.Region = arn.region
	}
	return res
}

// buildDevice describes the scanned asset as an endpoint.
func buildDevice(asset *inventory.Asset, version ocsf.Version) *ocsf.Device {
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
	res.Type = deviceTypeName(res.TypeID, platform)

	if osType := osTypeOf(platform); osType != ocsf.OSTypeUnknown {
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

// deviceTypeName is the string sibling of device.type_id. For a device that
// is none of OCSF's known types the sibling carries what cnspec calls the
// platform, which is more useful than the word "Other" and is what OCSF asks for.
func deviceTypeName(typeID int, platform *inventory.Platform) string {
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

// osTypeOf detects the operating system family of a platform. Assets that are
// not operating systems (cloud APIs, SaaS, Kubernetes objects) have none.
func osTypeOf(platform *inventory.Platform) int {
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

// buildCloud fills in the cloud environment of an asset. It returns nil for
// assets that are not cloud resources, which keeps the cloud profile off those
// events.
func buildCloud(asset *inventory.Asset, version ocsf.Version) *ocsf.Cloud {
	provider := cloudProvider(asset)
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
			// A GCP project is the account of a GCP asset, and account.uid is where
			// OCSF wants it. project_uid used to be the only home for it, but 1.9
			// deprecates the attribute in favor of account.uid -- a validator warns
			// on every event of a GCP asset that still carries it, so it is gated
			// the same way compliance.status_detail and device_hw_info.cpu_type
			// are. Setting the account regardless is what keeps the project id in
			// the document at 1.9, where project_uid is gone.
			res.Account = &ocsf.Account{UID: project, Type: "GCP Project"}
			if !version.AtLeast(ocsf.Version190) {
				res.ProjectUID = project
			}
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

// cloudProvider reports the cloud an asset belongs to, using the same
// vocabulary OCSF consumers expect ("AWS", "Azure", "GCP").
func cloudProvider(asset *inventory.Asset) string {
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
