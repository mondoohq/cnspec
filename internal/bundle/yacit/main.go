package main

import (
	"os"

	yacit "go.mondoo.com/cnspec/internal/yac-it"
	"go.mondoo.com/cnspec/policy"
)

//go:generate go run ./main.go
// Note: you still have to gofmt + goimports

func main() {
	res := yacit.New(yacit.YacItConfig{
		SkipUnmarshal: []string{
			"Impact", "ImpactValue", "Filters", "Remediation",
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
			"Filters": 80,
			"Checks":  70,

			// used in may structs
			"Uid":   100,
			"Mrn":   99,
			"Name":  98,
			"Title": 98,

			// policy & queries
			"Version": 97,
			"Impact":  97,
			"License": 96,
			"Tags":    70,
			"Mql":     60,

			"Authors": 51,

			"Docs":   50,
			"Refs":   49,
			"Groups": 40,

			// author
			"Email": 97,
		},
	})

	res.Add(&policy.Bundle{})
	res.Add(&policy.DeprecatedV7_Bundle{})

	os.WriteFile("../bundle.yac.go", []byte(res.String()), 0o644)
}
