// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package report_conversion

import (
	"os"
	"testing"

	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream/fex"
)

// AssertClean is the standard converter test: it runs conv on the report file at
// path and asserts it produces at least one document and that every document
// passes Validate. A converter is "done" when a real sample report converts clean
// — there are no golden files to maintain.
func AssertClean(t *testing.T, conv Converter, path string) []*fex.FindingDocument {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	docs, err := conv(data)
	if err != nil {
		t.Fatalf("convert %s: %v", path, err)
	}
	if len(docs) == 0 {
		t.Fatalf("convert %s: produced no documents", path)
	}
	for i, d := range docs {
		if err := Validate(d); err != nil {
			t.Errorf("document %d not clean: %v", i, err)
		}
	}
	return docs
}
