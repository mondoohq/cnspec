package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cnquery_config "go.mondoo.com/cnquery/apps/cnquery/cmd/config"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/stringx"
	"go.mondoo.com/cnquery/upstream"
	"go.mondoo.com/cnspec/apps/cnspec/cmd/fmtbundle"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/ranger-rpc"
)

func init() {
	// policy init
	policyBundlesCmd.AddCommand(policyInitCmd)

	// validate
	policyBundlesCmd.AddCommand(policyValidateCmd)

	// fmt
	policyBundlesCmd.AddCommand(policyFmtCmd)

	// bundle add
	policyUploadCmd.Flags().String("policy-version", "", "Override the version of each policy in the bundle.")
	policyBundlesCmd.AddCommand(policyUploadCmd)

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

func validate(policyBundle *policy.Bundle) []string {
	errors := []string{}

	// check that we have uids for policies and queries
	for i := range policyBundle.Policies {
		policy := policyBundle.Policies[i]
		policyId := strconv.Itoa(i)

		if policy.Uid == "" {
			errors = append(errors, fmt.Sprintf("policy %s does not define a UID", policyId))
		} else {
			policyId = policy.Uid
		}

		if policy.Name == "" {
			errors = append(errors, fmt.Sprintf("policy %s does not define a name", policyId))
		}
	}

	for j := range policyBundle.Queries {
		query := policyBundle.Queries[j]
		queryId := strconv.Itoa(j)
		if query.Uid == "" {
			errors = append(errors, fmt.Sprintf("query %s does not define a UID", queryId))
		} else {
			queryId = query.Uid
		}

		if query.Title == "" {
			errors = append(errors, fmt.Sprintf("query %s does not define a name", queryId))
		}
	}

	// we compile after the checks because it removes the uids and replaces it with mrns
	_, err := policyBundle.Compile(context.Background(), nil)
	if err != nil {
		log.Fatal().Err(err).Msg("could not validate policy bundle")
	}

	return errors
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

		errors := validate(policyBundle)
		if len(errors) > 0 {
			log.Error().Msg("could not validate policy bundle")
			for i := range errors {
				fmt.Fprintf(os.Stderr, stringx.Indent(2, errors[i]))
			}
			os.Exit(1)
		}
		log.Info().Msg("valid policy bundle")
	},
}

func formatPath(mqlBundlePath string) error {
	log.Info().Str("file", mqlBundlePath).Msg("format policy bundle(s)")
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
			return err
		}
	}
	return nil
}

func formatFile(filename string) error {
	log.Info().Str("file", filename).Msg("format file")
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
	Short:   "Apply style formatting to policy bundles.",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		for _, path := range args {
			err := formatPath(path)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

		}
		log.Info().Msg("completed formatting policy bundle(s)")
	},
}

var policyUploadCmd = &cobra.Command{
	Use:   "upload [path]",
	Short: "Add a user-owned policy to Mondoo Query Hub.",
	Args:  cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("policy-version", cmd.Flags().Lookup("policy-version"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		opts, optsErr := cnquery_config.ReadConfig()
		if optsErr != nil {
			log.Fatal().Err(optsErr).Msg("could not load configuration")
		}
		config.DisplayUsedConfig()

		filename := args[0]
		log.Info().Str("file", filename).Msg("load policy bundle")
		policyBundle, err := policy.BundleFromPaths(filename)
		if err != nil {
			log.Fatal().Err(err).Msg("could not load policy bundle")
		}

		errors := validate(policyBundle)
		if len(errors) > 0 {
			log.Error().Msg("could not validate policy bundle")
			for i := range errors {
				fmt.Fprintf(os.Stderr, stringx.Indent(2, errors[i]))
			}
			os.Exit(1)
		}
		log.Info().Msg("valid policy bundle")

		// compile manipulates the bundle, therefore we read it again
		policyBundle, err = policy.BundleFromPaths(filename)
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

		certAuth, _ := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		queryHubServices, err := policy.NewPolicyHubClient(opts.UpstreamApiEndpoint(), ranger.DefaultHttpClient(), certAuth)
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
