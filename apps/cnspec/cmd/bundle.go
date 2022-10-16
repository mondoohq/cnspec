package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.mondoo.com/cnspec/apps/cnspec/cmd/fmtbundle"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnspec/policy"
)

func init() {
	// policy init
	policyBundlesCmd.AddCommand(policyInitCmd)

	// validate
	policyBundlesCmd.AddCommand(policyValidateCmd)

	// fmt
	policyBundlesCmd.AddCommand(policyFmtCmd)

	rootCmd.AddCommand(policyBundlesCmd)
}

var policyBundlesCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Manage policy bundles",
}

//go:embed policy-example.mql.yaml
var embedPolicyTemplate []byte

var policyInitCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Create an example policy bundle that can be used as a starting point. If no filename is provided, `example-policy.mql.yml` is used",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := "example-policy.mql.yaml"
		if len(args) == 1 {
			name = args[0]
		}

		_, err := os.Stat(name)
		if err == nil {
			log.Fatal().Msgf("Policy '%s' already exists", name)
		}

		err = os.WriteFile(name, embedPolicyTemplate, 0o640)
		if err != nil {
			log.Fatal().Err(err).Msgf("Could not write '%s'", name)
		}
		log.Info().Msgf("Example policy file written to %s", name)
	},
}

var policyValidateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validates a policy bundle",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Str("file", args[0]).Msg("validate policy bundle")
		policyBundle, err := policy.BundleFromPaths(args[0])
		if err != nil {
			log.Fatal().Err(err).Msg("could not load policy bundle")
		}

		_, err = policyBundle.Compile(context.Background(), nil)
		if err != nil {
			log.Fatal().Err(err).Msg("could not validate policy bundle")
		}
		log.Info().Msg("valid policy bundle")
	},
}

func formatFile(filename string) error {
	log.Info().Str("filename", filename).Msg("format file")
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	bundle, err := fmtbundle.ParseYaml(data)
	if err != nil {
		return err
	}

	data, err = fmtbundle.Format(bundle)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, data, 0o644)
	if err != nil {
		return err
	}

	return nil
}

var policyFmtCmd = &cobra.Command{
	Use:     "format [path]",
	Aliases: []string{"fmt"},
	Short:   "Apply style formatting to policy bundles",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Str("file", args[0]).Msg("format policy bundle(s)")

		mqlBundlePath := args[0]
		fi, err := os.Stat(mqlBundlePath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if fi.IsDir() {
			filepath.WalkDir(mqlBundlePath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				// we ignore nested directories
				if d.IsDir() {
					return nil
				}

				// only consider .yaml|.yml files
				if strings.HasSuffix(d.Name(), ".mql.yaml") {
					err := formatFile(path)
					if err != nil {
						fmt.Println(err)
						os.Exit(1)
					}
				}
				return nil
			})
		} else {
			err := formatFile(mqlBundlePath)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
		log.Info().Msg("completed formatting policy bundle(s)")
	},
}
