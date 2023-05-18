package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cnquery_cmd "go.mondoo.com/cnquery/apps/cnquery/cmd"
	cnquery_config "go.mondoo.com/cnquery/apps/cnquery/cmd/config"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/upstream"
	"go.mondoo.com/cnspec/internal/bundle"
	"go.mondoo.com/cnspec/policy"
)

func init() {
	// policy init
	policyBundlesCmd.AddCommand(policyInitCmd)

	// validate
	policyLintCmd.Flags().StringP("output", "o", "cli", "Set output format: compact, sarif")
	policyLintCmd.Flags().String("output-file", "", "Set output file")
	policyBundlesCmd.AddCommand(policyLintCmd)

	// fmt
	policyBundlesCmd.AddCommand(policyFmtCmd)

	// docs
	policyDocsCmd.Flags().Bool("no-code", false, "enable/disable code blocks inside of docs")
	policyDocsCmd.Flags().Bool("no-ids", false, "enable/disable the printing of ID fields")
	policyBundlesCmd.AddCommand(policyDocsCmd)

	// publish
	policyPublishCmd.Flags().Bool("no-lint", false, "Disable linting of the bundle before publishing.")
	policyPublishCmd.Flags().String("policy-version", "", "Override the version of each policy in the bundle.")
	policyBundlesCmd.AddCommand(policyPublishCmd)

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
	Short: "Create an example policy bundle that you can use as a starting point. If you don't provide a filename, cnspec uses `example-policy.mql.yml`.",
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

var policyLintCmd = &cobra.Command{
	Use:     "lint [path]",
	Aliases: []string{"validate"},
	Short:   "Lint a policy bundle",
	Args:    cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("output", cmd.Flags().Lookup("output"))
		viper.BindPFlag("output-file", cmd.Flags().Lookup("output-file"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Str("file", args[0]).Msg("lint policy bundle")

		files, err := policy.WalkPolicyBundleFiles(args[0])
		if err != nil {
			log.Fatal().Err(err).Msg("could not find bundle files")
		}

		result, err := bundle.Lint(files...)
		if err != nil {
			log.Fatal().Err(err).Msg("could not lint bundle files")
		}

		out := os.Stdout
		if viper.GetString("output-file") != "" {
			out, err = os.Create(viper.GetString("output-file"))
			if err != nil {
				log.Fatal().Err(err).Msg("could not create output file")
			}
			defer out.Close()
		}

		switch viper.GetString("output") {
		case "cli":
			out.Write(result.ToCli())
		case "sarif":
			data, err := result.ToSarif(filepath.Dir(args[0]))
			if err != nil {
				log.Fatal().Err(err).Msg("could not generate sarif report")
			}
			out.Write(data)
		}

		if viper.GetString("output-file") == "" {
			if result.HasError() {
				log.Fatal().Msg("invalid policy bundle")
			} else {
				log.Info().Msg("valid policy bundle")
			}
		}
	},
}

var policyFmtCmd = &cobra.Command{
	Use:     "format [path]",
	Aliases: []string{"fmt"},
	Short:   "Apply style formatting to policy bundles.",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		for _, path := range args {
			err := bundle.FormatRecursive(path)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

		}
		log.Info().Msg("completed formatting policy bundle(s)")
	},
}

var policyPublishCmd = &cobra.Command{
	Use:     "publish [path]",
	Aliases: []string{"upload"},
	Short:   "Add a user-owned policy to Mondoo Query Hub.",
	Args:    cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("policy-version", cmd.Flags().Lookup("policy-version"))
		viper.BindPFlag("no-lint", cmd.Flags().Lookup("no-lint"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		opts, optsErr := cnquery_config.ReadConfig()
		if optsErr != nil {
			log.Fatal().Err(optsErr).Msg("could not load configuration")
		}
		config.DisplayUsedConfig()

		filename := args[0]
		log.Info().Str("file", filename).Msg("load policy bundle")
		files, err := policy.WalkPolicyBundleFiles(args[0])
		if err != nil {
			log.Fatal().Err(err).Msg("could not find bundle files")
		}

		noLint := viper.GetBool("no-lint")
		if !noLint {
			result, err := bundle.Lint(files...)
			if err != nil {
				log.Fatal().Err(err).Msg("could not lint bundle files")
			}

			// render cli output
			os.Stdout.Write(result.ToCli())

			if result.HasError() {
				log.Fatal().Msg("invalid policy bundle")
			} else {
				log.Info().Msg("valid policy bundle")
			}
		}

		// compile manipulates the bundle, therefore we read it again
		policyBundle, err := policy.BundleFromPaths(filename)
		if err != nil {
			log.Fatal().Err(err).Msg("could not load policy bundle")
		}

		log.Info().Str("space", opts.SpaceMrn).Msg("add policy bundle to space")
		overrideVersionFlag := false
		overrideVersion := viper.GetString("policy-version")
		if len(overrideVersion) > 0 {
			overrideVersionFlag = true
		}

		serviceAccount := opts.GetServiceCredential()
		if serviceAccount == nil {
			log.Fatal().Msg("cnquery has no credentials. Log in with `cnquery login`")
		}

		certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		if err != nil {
			log.Error().Err(err).Msg(errorMessageServiceAccount)
			os.Exit(cnquery_cmd.ConfigurationErrorCode)
		}

		httpClient, err := opts.GetHttpClient()
		if err != nil {
			log.Fatal().Err(err).Msg("error while creating Mondoo API client")
		}
		queryHubServices, err := policy.NewPolicyHubClient(opts.UpstreamApiEndpoint(), httpClient, certAuth)
		if err != nil {
			log.Fatal().Err(err).Msg("could not connect to policy hub")
		}

		// set the owner mrn for spaces
		policyBundle.OwnerMrn = opts.SpaceMrn
		ctx := context.Background()

		// override version and/or labels
		for i := range policyBundle.Policies {
			p := policyBundle.Policies[i]

			// override policy version
			if overrideVersionFlag {
				p.Version = overrideVersion
			}
		}

		// send data upstream
		_, err = queryHubServices.SetBundle(ctx, policyBundle)
		if err != nil {
			log.Fatal().Err(err).Msg("could not add policy bundle")
		}

		log.Info().Msg("successfully added policies")
	},
}

var policyDocsCmd = &cobra.Command{
	Use:     "docs [path]",
	Aliases: []string{},
	Short:   "Retrieve only the docs for a bundle.",
	Args:    cobra.MinimumNArgs(1),
	Hidden:  true,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("no-ids", cmd.Flags().Lookup("no-ids"))
		viper.BindPFlag("no-code", cmd.Flags().Lookup("no-code"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		bundle, err := policy.BundleFromPaths(args...)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to load bundle")
		}

		noIDs := viper.GetBool("no-ids")
		noCode := viper.GetBool("no-code")
		bundle.ExtractDocs(os.Stdout, noIDs, noCode)
	},
}
