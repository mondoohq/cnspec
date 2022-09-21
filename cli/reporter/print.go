package reporter

import "strings"

//go:generate protoc --proto_path=../../:. --go_out=. --go_opt=paths=source_relative  reporter.proto

type Format byte

const (
	Compact Format = iota + 1
	Summary
	Full
	Report
	YAML
	JSON
	JUnit
	CSV
)

// Formats that are supported by the reporter
var Formats = map[string]Format{
	"compact": Compact,
	"summary": Summary,
	"full":    Full,
	"":        Compact,
	"report":  Report,
	"yaml":    YAML,
	"yml":     YAML,
	"json":    JSON,
	"junit":   JUnit,
	"csv":     CSV,
}

func AllFormats() string {
	var res []string
	for k := range Formats {
		if k != "" && // default if nothing is provided, ignore
			k != "yml" { // don't show both yaml and yml
			res = append(res, k)
		}
	}
	return strings.Join(res, ", ")
}
