// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package aibom

import (
	"encoding/json"
	"io"
)

type JSONFormatter struct{}

func (f *JSONFormatter) Render(w io.Writer, bom *AiBom) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(bom)
}
