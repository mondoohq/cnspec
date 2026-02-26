// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package internal

import "go.mondoo.com/cnspec/v13/policy"

func mockFinding() *policy.FindingDocument {
	return &policy.FindingDocument{
		Finding: &policy.FindingDocument_Fex{
			Fex: &policy.FindingExchange{
				Id:      "scanme.nmap.org-tcp-22",
				Status:  policy.FindingStatus_FINDING_STATUS_AFFECTED,
				Summary: "Port 22 is open (ssh)",
				Details: &policy.FindingDetail{
					Category:    policy.FindingDetail_CATEGORY_SECURITY,
					Description: "A ssh service was detected running OpenSSH version 6.6.1p1 Ubuntu 2ubuntu2.13 on Linux. The service was identified through probed with very high confidence.",
					Severity: &policy.FindingSeverity{
						Rating: policy.SeverityRating_SEVERITY_RATING_CRITICAL,
					},
				},
				Affects: []*policy.Affects{
					{
						Component: &policy.Component{
							Identifiers: map[string]string{
								"hostname": "defender-scanning-vm",
							},
						},
					},
				},
				Evidences: []*policy.FindingEvidence{
					{
						Details: &policy.FindingEvidence_Connection{
							Connection: &policy.FindingConnection{
								DestinationAddress: "45.33.32.156",
								DestinationPort:    22,
								Protocol:           policy.FindingConnection_TCP,
							},
						},
					},
				},
				Source: &policy.FindingSource{
					Name: "VEX_SOURCE_NMAP",
				},
			},
		},
	}
}
