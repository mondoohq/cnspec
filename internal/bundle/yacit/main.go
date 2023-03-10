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
	})

	res.Add(&policy.Bundle{})
	res.Add(&policy.DeprecatedV7_Bundle{})

	os.WriteFile("../bundle.yac.go", []byte(res.String()), 0o644)
}
