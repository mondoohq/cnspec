// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package generator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	cr "go.mondoo.com/cnquery/v12/cli/reporter"
	"go.mondoo.com/cnquery/v12/mrn"
	"go.mondoo.com/cnquery/v12/utils/sortx"
	"go.mondoo.com/cnspec/v12"
	"go.mondoo.com/cnspec/v12/internal/sbom"
)

var LABEL_KERNEL_RUNNING = "mondoo.com/os/kernel-running"

// GenerateBom generates a BOM from a cnspec json report collection
func GenerateBom(r *cr.Report) []*sbom.Sbom {
	if r == nil {
		return nil
	}

	generator := &sbom.Generator{
		Vendor:  "Mondoo, Inc.",
		Name:    "cnspec",
		Version: cnspec.Version,
		Url:     "https://mondoo.com",
	}
	now := time.Now().UTC().Format(time.RFC3339)

	boms := []*sbom.Sbom{}
	for mrn := range r.Assets {
		asset := r.Assets[mrn]
		bom := &sbom.Sbom{
			Generator: generator,
			Timestamp: now,
			Status:    sbom.Status_STATUS_SUCCEEDED,
		}

		bom.Asset = &sbom.Asset{
			Name:        asset.GetName(),
			PlatformIds: nil,
			Platform:    &sbom.Platform{},
			Labels:      asset.GetLabels(),
			ExternalIds: []*sbom.ExternalID{},
			TraceId:     asset.GetTraceId(),
		}

		bom.Packages = []*sbom.Package{}

		// extract os packages and python packages
		dataPoints := r.Data[mrn]
		if dataPoints == nil {
			bom.Status = sbom.Status_STATUS_FAILED
			bom.ErrorMessage = "no data points found"
			boms = append(boms, bom)
			continue
		}
		// ensure deterministic order of enumeration
		keys := sortx.Keys(dataPoints.Values)
		for _, k := range keys {
			dataValue := dataPoints.Values[k]
			jsondata, err := cr.JsonValue(dataValue.Content)
			if err != nil {
				bom.Status = sbom.Status_STATUS_FAILED
				bom.ErrorMessage = errors.Wrap(err, "failed to parse json data").Error()
				continue
			}
			rb := BomFields{}
			err = json.Unmarshal(jsondata, &rb)
			if err != nil {
				bom.Status = sbom.Status_STATUS_FAILED
				bom.ErrorMessage = errors.Wrap(err, "failed to parse bom fields json data").Error()
				continue
			}
			if rb.Asset != nil {
				bom.Asset.Name = rb.Asset.Name
				bom.Asset.Platform.Name = rb.Asset.Platform
				bom.Asset.Platform.Version = rb.Asset.Version
				bom.Asset.Platform.Build = rb.Asset.Build
				bom.Asset.Platform.Family = rb.Asset.Family
				bom.Asset.Platform.Arch = rb.Asset.Arch
				bom.Asset.Platform.Cpes = rb.Asset.CPEs
				bom.Asset.Platform.Labels = rb.Asset.Labels
				bom.Asset.PlatformIds = enrichPlatformIds(rb.Asset.IDs)
				bom.Asset.Platform.Title = rb.Asset.PlatformTitle
			}

			if bom.Asset == nil {
				bom.Asset = &sbom.Asset{}
			}
			if bom.Asset.Labels == nil {
				bom.Asset.Labels = map[string]string{}
			}

			// store version of running kernel
			for _, kernel := range rb.KernelInstalled {
				if kernel.Running {
					bom.Asset.Labels[LABEL_KERNEL_RUNNING] = kernel.Version
				}
			}

			if rb.Packages != nil {
				for _, pkg := range rb.Packages {
					bomPkg := &sbom.Package{
						Name:         pkg.Name,
						Version:      pkg.Version,
						Architecture: pkg.Arch,
						Origin:       pkg.Origin,
						Purl:         pkg.Purl,
						Cpes:         pkg.CPEs,
						Type:         pkg.Format,
					}

					for _, filepath := range pkg.FilePaths {
						bomPkg.EvidenceList = append(bomPkg.EvidenceList, &sbom.Evidence{
							Type:  sbom.EvidenceType_EVIDENCE_TYPE_FILE,
							Value: filepath,
						})
					}

					bom.Packages = append(bom.Packages, bomPkg)
				}
			}

			for _, pkg := range rb.PythonPackages {
				bomPkg := &sbom.Package{
					Name:    pkg.Name,
					Version: pkg.Version,
					Purl:    pkg.Purl,
					Cpes:    pkg.CPEs,
					Type:    "pypi",
				}

				// deprecated path, all files are now in the FilePaths field
				// TODO: update once the python resource returns multiple results
				if pkg.FilePath != "" {
					bomPkg.EvidenceList = append(bomPkg.EvidenceList, &sbom.Evidence{
						Type:  sbom.EvidenceType_EVIDENCE_TYPE_FILE,
						Value: pkg.FilePath,
					})
				}

				for _, filepath := range pkg.FilePaths {
					bomPkg.EvidenceList = append(bomPkg.EvidenceList, &sbom.Evidence{
						Type:  sbom.EvidenceType_EVIDENCE_TYPE_FILE,
						Value: filepath,
					})
				}

				bom.Packages = append(bom.Packages, bomPkg)
			}

			for _, pkg := range rb.NpmPackages {
				bomPkg := &sbom.Package{
					Name:    pkg.Name,
					Version: pkg.Version,
					Purl:    pkg.Purl,
					Cpes:    pkg.CPEs,
					Type:    "npm",
				}

				for _, filepath := range pkg.FilePaths {
					bomPkg.EvidenceList = append(bomPkg.EvidenceList, &sbom.Evidence{
						Type:  sbom.EvidenceType_EVIDENCE_TYPE_FILE,
						Value: filepath,
					})
				}

				bom.Packages = append(bom.Packages, bomPkg)
			}
		}
		boms = append(boms, bom)
	}
	return boms
}

// enrichPlatformIds adds the platform id based on cnquery ids
// - AWS EC2 instance ARN
func enrichPlatformIds(ids []string) []string {
	platformIds := []string{}
	for i := range ids {
		platformIds = append(platformIds, ids[i])

		// handle AWS EC2 instance platform identifier and generate AWS ARN as additional identifier
		// EC2 arns have the following format arn:aws:ec2:<REGION>:<ACCOUNT_ID>:instance/<instance-id>
		// //platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/12345678910/regions/us-east-1/instances/i-1234567890abcdef0
		if strings.HasPrefix(ids[i], "//platformid.api.mondoo.app/runtime/aws/ec2/v1") {
			ec2mrn, err := mrn.NewMRN(ids[i])
			if err != nil {
				continue
			}

			accountID, _ := ec2mrn.ResourceID("accounts")
			region, _ := ec2mrn.ResourceID("regions")
			instanceID, _ := ec2mrn.ResourceID("instances")

			if accountID != "" && region != "" && instanceID != "" {
				platformIds = append(platformIds, fmt.Sprintf("arn:aws:ec2:%s:%s:instance/%s", region, accountID, instanceID))
			}
		}
	}
	return platformIds
}
