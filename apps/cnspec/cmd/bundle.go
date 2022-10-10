package cmd

import (
	"context"
	_ "embed"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnspec/policy"
)

func init() {
	// policy init
	policyBundlesCmd.AddCommand(policyInitCmd)

	// validate
	policyBundlesCmd.AddCommand(policyValidateCmd)

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
	Short: "Creates an example policy bundle that can be used as a starting point. If no filename is provided, `example-policy.mql.yml` us used",
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
