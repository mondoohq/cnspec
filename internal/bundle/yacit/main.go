// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"go/format"
	"os"

	yacit "go.mondoo.com/cnspec/v12/internal/yac-it"
	"go.mondoo.com/cnspec/v12/policy"
)

//go:generate go run ./main.go
// Note: you still have to gofmt + goimports

func main() {
	res := yacit.New(yacit.YacItConfig{
		SkipUnmarshal: []string{
			"Impact", "ImpactValue", "Filters", "Remediation", "RiskMagnitude", "HumanTime",
		},
		Package: "bundle",
		// field names with sort weight
		FieldOrder: map[string]int{
			// bundle
			"OwnerMrn": 100,
			"Policies": 99,
			"Props":    70,
			"Queries":  60,

			// groups
			"Filters":  80,
			"Checks":   70,
			"Controls": 70,

			// used in many structs
			"Uid":   100,
			"Mrn":   99,
			"Name":  98,
			"Title": 98,

			// policy & queries
			"Version":  97,
			"Impact":   97,
			"License":  96,
			"Valid":    96,
			"Variants": 59,
			"Tags":     70,
			"Require":  69,
			"Mql":      60,

			"Authors": 51,

			"Docs":          50,
			"Refs":          49,
			"Groups":        40,
			"ScoringSystem": 39,

			// frameworks
			"FrameworkOwner":        90,
			"FrameworkDependencies": 89,

			// author
			"Email": 97,
		},
	})

	res.Add(&policy.Bundle{})

	code := res.String()
	formatted, err := format.Source([]byte(code))
	if err != nil {
		panic(err)
	}

	os.WriteFile("../bundle.yac.go", formatted, 0o644)
}
