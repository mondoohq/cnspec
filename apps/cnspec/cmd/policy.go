// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/muesli/termenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v9/cli/config"
	"go.mondoo.com/cnquery/v9/cli/theme"
	"go.mondoo.com/cnquery/v9/providers"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream/gql"
	"go.mondoo.com/cnspec/v9/internal/bundle"
	"go.mondoo.com/cnspec/v9/policy"
	cnspec_upstream "go.mondoo.com/cnspec/v9/upstream"
	"gopkg.in/yaml.v3"
	"k8s.io/utils/ptr"
)

const (
	PolicyMrnPrefix          = "//policy.api.mondoo.app"
	PrivatePoliciesMrnPrefix = "//policy.api.mondoo.app/spaces"
)

func init() {
	rootCmd.AddCommand(policyCmd)
	policyCmd.AddCommand(policyDeleteCmd)
	policyCmd.AddCommand(policyEnableCmd)
	policyCmd.AddCommand(policyDisableCmd)
	policyCmd.AddCommand(policyInitCmd)

	// list
	policyCmd.AddCommand(policyListCmd)
	policyListCmd.Flags().StringP("file", "f", "", "a local bundle file")

	// upload
	policyUploadCmd.Flags().Bool("no-lint", false, "Disable linting of the bundle before publishing.")
	policyUploadCmd.Flags().String("policy-version", "", "Override the version of each policy in the bundle.")
	policyCmd.AddCommand(policyUploadCmd)

	// lint
	policyLintCmd.Flags().StringP("output", "o", "cli", "Set output format: compact, sarif")
	policyLintCmd.Flags().String("output-file", "", "Set output file")
	policyCmd.AddCommand(policyLintCmd)

	// fmt
	policyFmtCmd.Flags().Bool("sort", false, "sort the bundle.")
	policyCmd.AddCommand(policyFmtCmd)

	// info
	policyInfoCmd.Flags().StringP("file", "f", "", "a local bundle file")
	policyCmd.AddCommand(policyInfoCmd)

	// download
	policyDownloadCmd.Flags().StringP("file", "f", "", "output file")
	policyCmd.AddCommand(policyDownloadCmd)

	// docs
	policyDocsCmd.Flags().Bool("no-code", false, "enable/disable code blocks inside of docs")
	policyDocsCmd.Flags().Bool("no-ids", false, "enable/disable the printing of ID fields")
	policyCmd.AddCommand(policyDocsCmd)
}

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage local and upstream policies.",
}

var policyListCmd = &cobra.Command{
	Use:   "list [-f bundle]",
	Short: "List enabled policies in the connected space.",
	Args:  cobra.MaximumNArgs(0),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlag("file", cmd.Flags().Lookup("file")); err != nil {
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		bundleFile := viper.GetString("file")
		var policies []*policy.Policy
		if bundleFile != "" {
			policyBundle, err := policy.DefaultBundleLoader().BundleFromPaths(bundleFile)
			if err != nil {
				return err
			}
			policies = policyBundle.Policies
		} else {
			opts, err := config.Read()
			if err != nil {
				return err
			}
			config.DisplayUsedConfig()

			mondooClient, err := getGqlClient(opts)
			if err != nil {
				return err
			}

			policies, err = cnspec_upstream.SearchPolicy(
				context.Background(), mondooClient, opts.GetParentMrn(), ptr.To(true), ptr.To(true), ptr.To(true))
			if err != nil {
				return err
			}
		}
		for _, policy := range policies {
			fmt.Println(policy.Name + " " + policy.Version)
			id := policy.Uid
			if policy.Mrn != "" {
				id = policy.Mrn
			}
			fmt.Println(termenv.String("  " + id).Foreground(theme.DefaultTheme.Colors.Disabled))
		}

		return nil
	},
}

var policyUploadCmd = &cobra.Command{
	Use:   "upload my.mql.yaml",
	Short: "Upload a policy to the connected space.",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlag("policy-version", cmd.Flags().Lookup("policy-version")); err != nil {
			return err
		}
		if err := viper.BindPFlag("no-lint", cmd.Flags().Lookup("no-lint")); err != nil {
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		bundleFile, err := policy.DefaultBundleLoader().BundleFromPaths(args[0])
		if err != nil {
			log.Error().Msgf("failed to upload policies: %s", err)
			os.Exit(1)
		}

		opts, err := config.Read()
		if err != nil {
			log.Error().Msgf("failed to get config: %s", err)
			os.Exit(1)
		}
		config.DisplayUsedConfig()

		if err := ensureProviders(); err != nil {
			log.Fatal().Err(err).Msg("could not initialize providers")
		}
		noLint := viper.GetBool("no-lint")
		if !noLint {
			files, err := policy.WalkPolicyBundleFiles(args[0])
			if err != nil {
				log.Fatal().Err(err).Msg("could not find bundle files")
			}

			runtime := providers.DefaultRuntime()
			result, err := bundle.Lint(runtime.Schema(), files...)
			if err != nil {
				log.Fatal().Err(err).Msg("could not lint bundle files")
			}

			// render cli output
			if _, err := os.Stdout.Write(result.ToCli()); err != nil {
				log.Fatal().Err(err).Msg("could not write output")
			}

			if result.HasError() {
				log.Fatal().Msg("invalid policy bundle")
			} else {
				log.Info().Msg("valid policy bundle")
			}
		}

		policyHub, err := getPolicyHubClient(opts)
		if err != nil {
			log.Error().Msgf("failed to create upstream client: %s", err)
			os.Exit(1)
		}

		// override policy version
		overrideVersion := viper.GetString("policy-version")
		if len(overrideVersion) > 0 {
			for i := range bundleFile.Policies {
				p := bundleFile.Policies[i]
				p.Version = overrideVersion
			}
		}

		ctx := context.Background()
		bundleFile.OwnerMrn = opts.GetParentMrn()
		_, err = policyHub.SetBundle(ctx, bundleFile)
		if err != nil {
			log.Error().Msgf("failed to upload policies: %s", err)
			os.Exit(1)
		}

		successMsg := fmt.Sprintf("successfully uploaded %d ", len(bundleFile.Policies))
		if len(bundleFile.Policies) > 1 {
			successMsg += "policies"
		} else {
			successMsg += "policy"
		}
		log.Info().Msg(theme.DefaultTheme.Success(successMsg))

		mondooClient, err := getGqlClient(opts)
		if err != nil {
			log.Error().Msgf("failed to create upstream client: %s", err)
			os.Exit(1)
		}

		for _, p := range bundleFile.Policies {
			fmt.Println("  policy: " + p.Name + " " + p.Version)
			fmt.Println(termenv.String("    " + getPolicyMrn(opts.GetParentMrn(), p.Uid)).Foreground(theme.DefaultTheme.Colors.Disabled))
		}

		space, err := cnspec_upstream.GetSpace(ctx, mondooClient, opts.GetParentMrn())
		if err != nil {
			log.Error().Msgf("failed to get space: %s", err)
			os.Exit(1)
		}

		fmt.Println("  space: " + space.Name)
		fmt.Println(termenv.String("    " + space.Mrn).Foreground(theme.DefaultTheme.Colors.Disabled))
	},
}

var policyDeleteCmd = &cobra.Command{
	Use:   "delete UID/MRN",
	Short: "Delete a policy from the connected space.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		opts, err := config.Read()
		if err != nil {
			log.Error().Msgf("failed to get config: %s", err)
			os.Exit(1)
		}
		config.DisplayUsedConfig()

		policyMrn := args[0]
		if !strings.HasPrefix(policyMrn, PolicyMrnPrefix) {
			policyMrn = getPolicyMrn(opts.GetParentMrn(), args[0])
		}

		// Add a meaningful error when trying to delete a public policy
		if !strings.HasPrefix(policyMrn, PrivatePoliciesMrnPrefix) {
			log.Error().Msgf("failed to delete policy: it is only possible to delete private policies")
			os.Exit(1)
		}

		policyHub, err := getPolicyHubClient(opts)
		if err != nil {
			log.Error().Msgf("failed to upload policies: %s", err)
			os.Exit(1)
		}

		ctx := context.Background()
		p, err := policyHub.GetPolicy(ctx, &policy.Mrn{Mrn: policyMrn})
		if err != nil {
			log.Error().Msgf("failed to get policy: %s", err)
			os.Exit(1)
		}

		_, err = policyHub.DeletePolicy(ctx, &policy.Mrn{Mrn: policyMrn})
		if err != nil {
			log.Error().Msgf("failed to delete policy: %s", err)
			os.Exit(1)
		}

		log.Info().Msg(theme.DefaultTheme.Success("successfully removed policy from space"))

		mondooClient, err := getGqlClient(opts)
		if err != nil {
			log.Error().Msgf("failed to create upstream client: %s", err)
			os.Exit(1)
		}

		space, err := cnspec_upstream.GetSpace(ctx, mondooClient, opts.GetParentMrn())
		if err != nil {
			log.Error().Msgf("failed to get space: %s", err)
			os.Exit(1)
		}

		fmt.Println("  policy: " + p.Name + " " + p.Version)
		fmt.Println(termenv.String("    " + p.Mrn).Foreground(theme.DefaultTheme.Colors.Disabled))

		fmt.Println("  space: " + space.Name)
		fmt.Println(termenv.String("    " + space.Mrn).Foreground(theme.DefaultTheme.Colors.Disabled))
	},
}

var policyInfoCmd = &cobra.Command{
	Use:     "info UID/MRN",
	Short:   "Show more info about a policy from the connected space.",
	Aliases: []string{"show"},
	Args:    cobra.MaximumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlag("file", cmd.Flags().Lookup("file")); err != nil {
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		bundleFile := viper.GetString("file")
		var policies []*policy.Policy
		if bundleFile != "" {
			policyBundle, err := policy.DefaultBundleLoader().BundleFromPaths(bundleFile)
			if err != nil {
				log.Error().Msgf("failed to read bundle: %s", err)
				os.Exit(1)
			}
			policies = policyBundle.Policies
		} else {
			opts, err := config.Read()
			if err != nil {
				log.Error().Msgf("failed to get config: %s", err)
				os.Exit(1)
			}
			config.DisplayUsedConfig()

			policyMrn := args[0]
			if !strings.HasPrefix(policyMrn, PolicyMrnPrefix) {
				policyMrn = getPolicyMrn(opts.GetParentMrn(), args[0])
			}

			policyHub, err := getPolicyHubClient(opts)
			if err != nil {
				log.Error().Msgf("failed to create upstream client: %s", err)
				os.Exit(1)
			}

			ctx := context.Background()
			p, err := policyHub.GetPolicy(ctx, &policy.Mrn{Mrn: policyMrn})
			if err != nil {
				log.Error().Msgf("failed to get policy: %s", err)
				os.Exit(1)
			}
			policies = append(policies, p)
		}

		for _, p := range policies {
			checks := map[string]struct{}{}
			queries := map[string]struct{}{}
			referenced := map[string]struct{}{}
			sections := []*policy.PolicyGroup{}
			for _, g := range p.Groups {
				if g.Type == policy.GroupType_CHAPTER || g.Type == policy.GroupType_UNCATEGORIZED {
					sections = append(sections, g)
				}
				for _, c := range g.Checks {
					checks[c.Mrn] = struct{}{}
				}
				for _, q := range g.Queries {
					queries[q.Mrn] = struct{}{}
				}
				for _, p := range g.Policies {
					referenced[p.Mrn] = struct{}{}
				}
			}

			fmt.Println("Name:        " + p.Name)
			fmt.Println("Version:     " + p.Version)
			if p.Uid != "" {
				fmt.Println("UID:         " + p.Uid)
			}
			if p.Mrn != "" {
				fmt.Println("MRN:         " + p.Mrn)
			}
			if len(p.Authors) > 0 {
				fmt.Printf("Authors:     %s <%s>\n", p.Authors[0].Name, p.Authors[0].Email)
				p.Authors = p.Authors[1:]
				for _, a := range p.Authors {
					fmt.Printf("          %s <%s>\n", a.Name, a.Email)
				}
			}
			if p.License != "" {
				fmt.Println("License:     " + p.License)
			} else {
				fmt.Println("License:     none")
			}
			if len(checks) > 0 {
				fmt.Printf("Checks:      %d\n", len(checks))
			}
			if len(queries) > 0 {
				fmt.Printf("Queries:     %d\n", len(queries))
			}
			if len(referenced) > 0 {
				fmt.Printf("Policies:    %d\n", len(referenced))
			}
			if p.Summary != "" {
				fmt.Println("Summary:     " + p.Summary)
			}
			if p.Docs != nil && p.Docs.Desc != "" {
				fmt.Println("Description:")
				fmt.Println(p.Docs.Desc)
			}
			if len(sections) > 0 {
				fmt.Println("Sections:")
				for i, s := range sections {
					fmt.Printf("  %d. %s\n", i+1, s.Title)
				}
			}
			fmt.Println()
			fmt.Println()
		}
	},
}

var policyDownloadCmd = &cobra.Command{
	Use:   "download UID/MRN",
	Short: "download a policy to a local bundle file.",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlag("file", cmd.Flags().Lookup("file")); err != nil {
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		outputFile := viper.GetString("file")
		if outputFile == "" {
			log.Error().Msgf("output file is required")
			os.Exit(1)
		}

		opts, err := config.Read()
		if err != nil {
			log.Error().Msgf("failed to get config: %s", err)
			os.Exit(1)
		}
		config.DisplayUsedConfig()

		policyMrn := args[0]
		if !strings.HasPrefix(policyMrn, PolicyMrnPrefix) {
			policyMrn = getPolicyMrn(opts.GetParentMrn(), args[0])
		}

		policyHub, err := getPolicyHubClient(opts)
		if err != nil {
			log.Error().Msgf("failed to create upstream client: %s", err)
			os.Exit(1)
		}

		ctx := context.Background()
		p, err := policyHub.GetBundle(ctx, &policy.Mrn{Mrn: policyMrn})
		if err != nil {
			log.Error().Msgf("failed to download policy: %s", err)
			os.Exit(1)
		}

		data, err := yaml.Marshal(p)
		if err != nil {
			log.Error().Msgf("failed to marshal policy: %s", err)
			os.Exit(1)
		}
		if err := os.WriteFile(outputFile, data, 0o644); err != nil {
			log.Error().Msgf("failed to store policy: %s", err)
			os.Exit(1)
		}
		log.Info().Msg(theme.DefaultTheme.Success("successfully downloaded to ", outputFile))
	},
}

var policyEnableCmd = &cobra.Command{
	Use:   "enable UID/MRN",
	Short: "Enables a policy in the connected space.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		opts, err := config.Read()
		if err != nil {
			log.Error().Msgf("failed to get config: %s", err)
			os.Exit(1)
		}
		config.DisplayUsedConfig()

		policyMrn := args[0]
		if !strings.HasPrefix(policyMrn, PolicyMrnPrefix) {
			policyMrn = getPolicyMrn(opts.GetParentMrn(), args[0])
		}

		policyResolver, err := getPolicyResolverClient(opts)
		if err != nil {
			log.Error().Msgf("failed to create upstream client: %s", err)
			os.Exit(1)
		}

		ctx := context.Background()
		_, err = policyResolver.Assign(ctx, &policy.PolicyAssignment{
			AssetMrn:   opts.GetParentMrn(),
			PolicyMrns: []string{policyMrn},
		})
		if err != nil {
			log.Error().Msgf("failed to enable policy in space: %s", err)
			os.Exit(1)
		}
		log.Info().Msg(theme.DefaultTheme.Success("successfully enabled policy in space"))
	},
}

var policyDisableCmd = &cobra.Command{
	Use:   "disable UID/MRN",
	Short: "Disables a policy in the connected space.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		opts, err := config.Read()
		if err != nil {
			log.Error().Msgf("failed to get config: %s", err)
			os.Exit(1)
		}
		config.DisplayUsedConfig()

		policyMrn := args[0]
		if !strings.HasPrefix(policyMrn, PolicyMrnPrefix) {
			policyMrn = getPolicyMrn(opts.GetParentMrn(), args[0])
		}

		policyResolver, err := getPolicyResolverClient(opts)
		if err != nil {
			log.Error().Msgf("failed to create upstream client: %s", err)
			os.Exit(1)
		}

		ctx := context.Background()
		_, err = policyResolver.Unassign(ctx, &policy.PolicyAssignment{
			AssetMrn:   opts.GetParentMrn(),
			PolicyMrns: []string{policyMrn},
		})
		if err != nil {
			log.Error().Msgf("failed to disable policy to space: %s", err)
			os.Exit(1)
		}
		log.Info().Msg(theme.DefaultTheme.Success("successfully disabled policy in space"))
	},
}

//go:embed policy-example.mql.yaml
var embedPolicyTemplate []byte

var policyInitCmd = &cobra.Command{
	Use:     "init [path]",
	Short:   "Create an example policy bundle that you can use as a starting point. If you don't provide a filename, cnspec uses `example-policy.mql.yml`.",
	Aliases: []string{"new"},
	Args:    cobra.MaximumNArgs(1),
	Run:     runPolicyInit,
}

func runPolicyInit(cmd *cobra.Command, args []string) {
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
}

var policyFmtCmd = &cobra.Command{
	Use:     "format [path]",
	Aliases: []string{"fmt"},
	Short:   "Apply style formatting to one or more policy bundles.",
	Args:    cobra.MinimumNArgs(1),
	Run:     runPolicyFmt,
}

func runPolicyFmt(cmd *cobra.Command, args []string) {
	sort, _ := cmd.Flags().GetBool("sort")
	if err := ensureProviders(); err != nil {
		log.Fatal().Err(err).Msg("could not initialize providers")
	}
	for _, path := range args {
		err := bundle.FormatRecursive(path, sort)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	log.Info().Msg("completed formatting policy bundle(s)")
}

var policyLintCmd = &cobra.Command{
	Use:     "lint [path]",
	Aliases: []string{"validate"},
	Short:   "Lint a policy bundle.",
	Args:    cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlag("output", cmd.Flags().Lookup("output")); err != nil {
			return err
		}
		if err := viper.BindPFlag("output-file", cmd.Flags().Lookup("output-file")); err != nil {
			return err
		}
		return nil
	},
	Run: runPolicyLint,
}

func runPolicyLint(cmd *cobra.Command, args []string) {
	log.Info().Str("file", args[0]).Msg("lint policy bundle")
	if err := ensureProviders(); err != nil {
		log.Fatal().Err(err).Msg("could not initialize providers")
	}

	files, err := policy.WalkPolicyBundleFiles(args[0])
	if err != nil {
		log.Fatal().Err(err).Msg("could not find bundle files")
	}

	runtime := providers.DefaultRuntime()
	result, err := bundle.Lint(runtime.Schema(), files...)
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
		if _, err := out.Write(result.ToCli()); err != nil {
			log.Fatal().Err(err).Msg("could not write output")
		}
	case "sarif":
		data, err := result.ToSarif(filepath.Dir(args[0]))
		if err != nil {
			log.Fatal().Err(err).Msg("could not generate sarif report")
		}
		if _, err := out.Write(data); err != nil {
			log.Fatal().Err(err).Msg("could not write output")
		}
	}

	if viper.GetString("output-file") == "" {
		if result.HasError() {
			log.Fatal().Msg("invalid policy bundle")
		} else {
			log.Info().Msg("valid policy bundle")
		}
	}
}

var policyDocsCmd = &cobra.Command{
	Use:     "docs [path]",
	Aliases: []string{},
	Short:   "Retrieve only the docs for a bundle.",
	Args:    cobra.MinimumNArgs(1),
	Hidden:  true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlag("no-ids", cmd.Flags().Lookup("no-ids")); err != nil {
			return err
		}
		if err := viper.BindPFlag("no-code", cmd.Flags().Lookup("no-code")); err != nil {
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		bundleLoader := policy.DefaultBundleLoader()
		bundle, err := bundleLoader.BundleFromPaths(args...)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to load bundle")
		}

		noIDs := viper.GetBool("no-ids")
		noCode := viper.GetBool("no-code")
		bundle.ExtractDocs(os.Stdout, noIDs, noCode)
	},
}

func getGqlClient(opts *config.Config) (*gql.MondooClient, error) {
	serviceAccount := opts.GetServiceCredential()
	if serviceAccount == nil {
		return nil, fmt.Errorf("cnspec has no credentials. Log in with `cnspec login`")
	}

	httpClient, err := opts.GetHttpClient()
	if err != nil {
		return nil, err
	}

	upstreamConfig := &upstream.UpstreamConfig{
		SpaceMrn:    opts.GetParentMrn(),
		ApiEndpoint: opts.UpstreamApiEndpoint(),
		ApiProxy:    opts.APIProxy,
		Creds:       serviceAccount,
	}

	mondooClient, err := gql.NewClient(upstreamConfig, httpClient)
	if err != nil {
		return nil, err
	}

	return mondooClient, nil
}

func getPolicyMrn(spaceMrn, policyUid string) string {
	prefix := strings.Replace(spaceMrn, "captain", "policy", 1)
	return prefix + "/policies/" + policyUid
}

func getPolicyHubClient(opts *config.Config) (*policy.PolicyHubClient, error) {
	certAuth, err := upstream.NewServiceAccountRangerPlugin(opts.GetServiceCredential())
	if err != nil {
		return nil, err
	}

	httpClient, err := opts.GetHttpClient()
	if err != nil {
		return nil, err
	}
	return policy.NewPolicyHubClient(opts.UpstreamApiEndpoint(), httpClient, certAuth)
}

func getPolicyResolverClient(opts *config.Config) (*policy.PolicyResolverClient, error) {
	certAuth, err := upstream.NewServiceAccountRangerPlugin(opts.GetServiceCredential())
	if err != nil {
		return nil, err
	}

	httpClient, err := opts.GetHttpClient()
	if err != nil {
		return nil, err
	}
	return policy.NewPolicyResolverClient(opts.UpstreamApiEndpoint(), httpClient, certAuth)
}
