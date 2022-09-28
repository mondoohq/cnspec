package executor

import (
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/resources/packs/all/info"
)

func MustCompile(code string) *llx.CodeBundle {
	codeBundle, err := mqlc.Compile(code, info.Registry.Schema(), cnquery.DefaultFeatures, nil)
	if err != nil {
		panic(err)
	}
	return codeBundle
}

func MustGetOneDatapoint(codeBundle *llx.CodeBundle) string {
	if len(codeBundle.CodeV2.Entrypoints()) != 1 {
		panic("code bundle has more than 1 entrypoint")
	}

	entrypoint := codeBundle.CodeV2.Entrypoints()[0]
	checksum, ok := codeBundle.CodeV2.Checksums[entrypoint]
	if !ok {
		panic("could not find the data point for the entrypoint")
	}

	return checksum
}
