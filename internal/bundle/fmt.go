package bundle

import (
	"bytes"
	"os"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/policy"
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

// FormatRecursive iterates recursively through all .mql.yaml files and formats them
func FormatRecursive(mqlBundlePath string) error {
	log.Info().Str("file", mqlBundlePath).Msg("format policy bundle(s)")
	_, err := os.Stat(mqlBundlePath)
	if err != nil {
		return errors.New("file " + mqlBundlePath + " does not exist")
	}

	files, err := policy.WalkPolicyBundleFiles(mqlBundlePath)
	if err != nil {
		return err
	}

	for i := range files {
		f := files[i]
		err := FormatFile(f)
		if err != nil {
			return errors.Wrap(err, "could not format file: "+f)
		}
	}
	return nil
}

// Format formats the .mql.yaml bundle
func FormatFile(filename string) error {
	log.Info().Str("file", filename).Msg("format file")
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	b, err := ParseYaml(data)
	if err != nil {
		return err
	}

	data, err = Format(b)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, data, 0o644)
	if err != nil {
		return err
	}

	return nil
}
