package bundle

import (
	"bytes"
	"os"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnspec/policy"
	"gopkg.in/yaml.v3"
	k8s_yaml "sigs.k8s.io/yaml"
)

// Formats the given bundle to a yaml string
func Format[T any](bundle *T) ([]byte, error) {
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

// ParseYaml loads a yaml file and parse it into the go struct
func ParseYaml(data []byte) (*Bundle, error) {
	baseline := Bundle{}

	err := yaml.Unmarshal([]byte(data), &baseline)
	return &baseline, err
}

func DeprecatedV7_ToV8(data []byte) ([]byte, error) {
	// In the case of deprecated V7, we are only going to focus on the
	// conversion, throwing away everything else, including comments.
	// The focus is to get it to v8, none of the other formatting matters in this
	// step.
	v7baseline := policy.DeprecatedV7_Bundle{}
	if err := k8s_yaml.Unmarshal([]byte(data), &v7baseline); err != nil {
		return nil, err
	}

	v8 := v7baseline.ToV8()

	// this step will unfortunately not produce well-formatted YAML at all
	// because the proto structures don't have the yaml tags (only the
	// yac-it structures do) ...
	interim, err := Format(v8)
	if err != nil {
		return nil, err
	}

	// ... so we have to ping pong convert it a bit ...
	v8yaci, err := ParseYaml(interim)
	if err != nil {
		return nil, err
	}

	// ... until we have it where we want it
	return Format(v8yaci)
}

// Format formats the .mql.yaml bundle
func FormatFile(filename string) error {
	log.Info().Str("file", filename).Msg("format file")
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	b, err := ParseYaml(data)
	// we have v7 structs in v8 bundle, so it can happen that v7 parses properly
	// for that case we need to make sure all the structs are properly converted
	if err != nil || hasV7Structs(b) {
		data, err = DeprecatedV7_ToV8(data)
	} else {
		data, err = Format(b)
	}
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, data, 0o644)
	if err != nil {
		return err
	}

	return nil
}

func hasV7Structs(b *Bundle) bool {
	for i := range b.Policies {
		p := b.Policies[i]
		if len(p.Specs) > 0 {
			return true
		}
	}
	return false
}
