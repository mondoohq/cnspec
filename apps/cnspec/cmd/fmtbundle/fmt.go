package fmtbundle

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

// Formats the given bundle to a yaml string
func Format(bundle *PolicyBundle) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	err := enc.Encode(bundle)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
