// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/abiosoft/colima/util/terminal"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v11/cli/components"
	"go.mondoo.com/cnquery/v11/cli/config"
	"go.mondoo.com/cnquery/v11/cli/theme"
	"go.mondoo.com/cnspec/v11/internal/tfgen"
	cnspec_upstream "go.mondoo.com/cnspec/v11/upstream"
)

const (
	mondooProviderSource  = "mondoohq/mondoo"
	mondooProviderVersion = "~> 0.19"

	requiredTerraformVersion = ">= 1.0.11"
	installTerraformVersion  = "1.9.8"

	spacePrefix = "//captain.api.mondoo.app/spaces/"
)

func init() {
	// cnspec integrate
	rootCmd.AddCommand(integrateCmd)

	// global flags for the integrate command
	integrateCmd.PersistentFlags().String("space", "", "Set the space to create the integration")
	integrateCmd.PersistentFlags().String("output", "", "Location to write automation code")
	integrateCmd.PersistentFlags().String("integration-name", "", "The name of the integration")

	// cnspec integrate azure
	integrateCmd.AddCommand(integrateAzureCmd)
	integrateAzureCmd.Flags().String("subscription-id", "", "Azure subscription used to create resources")
}

var (
	integrateCmd = &cobra.Command{
		Use:     "integrate",
		Aliases: []string{"onboard"},
		Hidden:  true,
		Short:   "Onboard integrations for continuous scanning into the Mondoo platform",
		Long:    "Run automation code to onboard your account and deploy Mondoo into various environments.",
	}
	integrateAzureCmd = &cobra.Command{
		Use:     "azure",
		Aliases: []string{"az"},
		Short:   "Onboard Microsoft Azure",
		Long:    `Use this command to connect your Azure environment into the Mondoo platform.`,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := viper.BindPFlag("space", cmd.Flags().Lookup("space")); err != nil {
				return err
			}
			if err := viper.BindPFlag("output", cmd.Flags().Lookup("output")); err != nil {
				return err
			}
			if err := viper.BindPFlag("subscription-id", cmd.Flags().Lookup("subscription-id")); err != nil {
				return err
			}
			return viper.BindPFlag("integration-name", cmd.Flags().Lookup("integration-name"))
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			space, err := cmd.Flags().GetString("space")
			if err != nil {
				return err
			}
			subscriptionID, err := cmd.Flags().GetString("subscription-id")
			if err != nil {
				return err
			}
			output, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}
			integrationName, err := cmd.Flags().GetString("integration-name")
			if err != nil {
				return err
			}

			// Verify if space exists, which verifies we have access to the Mondoo platform
			opts, err := config.Read()
			if err != nil {
				return err
			}
			config.DisplayUsedConfig()
			mondooClient, err := getGqlClient(opts)
			if err != nil {
				return err
			}
			spaceInfo, err := cnspec_upstream.GetSpace(context.Background(), mondooClient, spacePrefix+space)
			if err != nil {
				log.Fatal().Msgf("unable to verify access to space '%s': %s", space, err)
			}
			log.Info().Msg("using space " + theme.DefaultTheme.Success(spaceInfo.Mrn))

			// Discover the subscription used to create resources in the cloud, if it wasn't specified. Note
			// that this will also verify that we have access to Azure. If we fail, we shouldn't try to continue.
			if subscriptionID == "" {
				azAccountJSON, err := exec.Command("az", "account", "list", "-o", "json").Output()
				if err != nil {
					return errors.Wrap(err, "unable to detect Azure subscriptions")
				}
				var azAccounts []azAccount
				if err := json.Unmarshal(azAccountJSON, &azAccounts); err != nil {
					return err
				}

				isTTY := isatty.IsTerminal(os.Stdout.Fd())
				if isTTY {
					selected := components.Select(
						"Select the primary subscription where resources will be created",
						azAccounts,
					)
					if selected >= 0 {
						subscriptionID = azAccounts[selected].ID
					}
				} else {
					fmt.Println(components.List(theme.OperatingSystemTheme, azAccounts))
					log.Fatal().
						Msg("cannot continue, missing subscription id, use --subscription-id to select a subscription")
				}
			}

			if subscriptionID == "" {
				log.Error().Msg("no subscription selected")
				os.Exit(1)
			}

			// Validate integration name is not empty, if it is, generate a random one
			if integrationName == "" {
				integrationName = generateIntegrationName(subscriptionID)
				log.Info().Msg("integration name not provided, using " + theme.DefaultTheme.Primary(integrationName))
			}

			// TODO ideally, we should verify that the user has the "Privileged Role Administrator" or "Global Administrator"
			// => https://learn.microsoft.com/en-us/entra/identity/role-based-access-control/permissions-reference#privileged-role-administrator

			// Generate HCL for azure deployment
			log.Info().Msg("generating automation code")
			hcl, err := GenerateAzureHCL(space, subscriptionID, integrationName)
			if err != nil {
				return errors.Wrap(err, "unable to generate automation code")
			}

			// Write generated code to disk
			dirname, err := WriteHCL(hcl, output, "azure")
			if err != nil {
				return err
			}
			log.Info().Msgf("code stored at %s", theme.DefaultTheme.Secondary(dirname))

			// Run Terraform
			return TerraformPlanAndExecute(dirname, space)
		},
	}
)

func GenerateAzureHCL(space, subscriptionID, integrationName string) (string, error) {
	requiredProvidersBlock, err := tfgen.CreateRequiredProviders(
		tfgen.NewRequiredProvider("mondoo",
			tfgen.HclRequiredProviderWithSource(mondooProviderSource),
			tfgen.HclRequiredProviderWithVersion(mondooProviderVersion),
		),
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate required providers")
	}

	featuresBlock, err := tfgen.HclCreateGenericBlock("features", []string{}, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to append features block to azurerm provider")
	}
	selfSignedCertSubjectBlock, err := tfgen.HclCreateGenericBlock("subject", nil,
		tfgen.Attributes{"common_name": "mondoo"},
	)
	if err != nil {
		return "", errors.Wrap(err, "failed lfkjaslfkr")
	}

	var (
		providerAzureAD = tfgen.NewProvider("azuread")
		providerAzureRM = tfgen.NewProvider("azurerm",
			tfgen.HclProviderWithAttributes(tfgen.Attributes{"subscription_id": subscriptionID}),
			tfgen.HclProviderWithGenericBlocks(featuresBlock),
		)
		providerMondoo = tfgen.NewProvider("mondoo", tfgen.HclProviderWithAttributes(
			tfgen.Attributes{"space": space},
		))
		dataADClientConfig    = tfgen.NewDataSource("azuread_client_config", "current")
		resourceAdApplication = tfgen.NewResource("azuread_application", "mondoo",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"display_name":  "mondoo_security", // @afiune should we customize this?
				"owners":        []interface{}{dataADClientConfig.TraverseRef("object_id")},
				"marketing_url": "https://www.mondoo.com/",
			}),
		)
		resourceTLSPrivateKey = tfgen.NewResource("tls_private_key", "credential",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{"algorithm": "RSA"}),
		)
		resourceTLSSelfSignedCert = tfgen.NewResource("tls_self_signed_cert", "credential",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"private_key_pem":       resourceTLSPrivateKey.TraverseRef("private_key_pem"),
				"validity_period_hours": 4096,
				"early_renewal_hours":   3,
				"allowed_uses": []string{
					"key_encipherment",
					"digital_signature",
					"data_encipherment",
					"cert_signing",
				},
			}),
			tfgen.HclResourceWithGenericBlocks(selfSignedCertSubjectBlock),
		)
		resourceADApplicationCertificate = tfgen.NewResource("azuread_application_certificate", "mondoo",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				// see https://github.com/hashicorp/terraform-provider-azuread/issues/1227
				"application_id": resourceAdApplication.TraverseRef("id"),
				"type":           "AsymmetricX509Cert",
				"value":          resourceTLSSelfSignedCert.TraverseRef("cert_pem"),
			}),
		)
		resourceADServicePrincipal = tfgen.NewResource("azuread_service_principal", "mondoo",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"client_id": resourceAdApplication.TraverseRef("client_id"),
				"owners":    []interface{}{dataADClientConfig.TraverseRef("object_id")},
			}),
		)
		dataRMAllSubscriptions         = tfgen.NewDataSource("azurerm_subscriptions", "available")
		resourceRMReaderRoleAssignment = tfgen.NewResource("azurerm_role_assignment", "reader",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"role_definition_name": "Reader",

				"count":        tfgen.NewFuncCall("length", dataRMAllSubscriptions.TraverseRef("subscriptions")),
				"scope":        dataRMAllSubscriptions.TraverseRef("subscriptions[count.index]", "id"),
				"principal_id": resourceADServicePrincipal.TraverseRef("object_id"),
			}),
		)
		// This is the way we avoid Grant Admin Consent issue.
		//
		// => https://docs.microsoft.com/en-us/azure/active-directory/roles/permissions-reference#directory-readers
		//
		resourceADReadersDirectoryRole = tfgen.NewResource("azuread_directory_role", "readers",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{"display_name": "Directory Readers"}),
		)
		resourceTimeSleep = tfgen.NewResource("time_sleep", "wait_time",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{"create_duration": "60s"}),
		)
		resourceADReadersRoleAssignment = tfgen.NewResource("azuread_directory_role_assignment", "readers",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"role_id":             resourceADReadersDirectoryRole.TraverseRef("template_id"),
				"principal_object_id": resourceADServicePrincipal.TraverseRef("object_id"),
				"depends_on":          []interface{}{resourceTimeSleep.TraverseRef()},
			}),
		)
		resourceMondooIntegration = tfgen.NewResource("mondoo_integration_azure", "this",
			tfgen.HclResourceWithAttributes(tfgen.Attributes{
				"name":      integrationName,
				"tenant_id": dataADClientConfig.TraverseRef("tenant_id"),
				"client_id": resourceAdApplication.TraverseRef("client_id"),
				"scan_vms":  false, // @afiune should be a parameter and we need a custom role
				"credentials": tfgen.Attributes{
					// TODO @afiune support
					"pem_file": tfgen.NewFuncCall("join",
						tfgen.CreateSimpleTraversal(`"\n", [tls_self_signed_cert.credential.cert_pem, tls_private_key.credential.private_key_pem]`),
					),
				},
				"depends_on": []interface{}{
					resourceADServicePrincipal.TraverseRef(),
					resourceRMReaderRoleAssignment.TraverseRef(),
					resourceADApplicationCertificate.TraverseRef(),
					resourceADReadersRoleAssignment.TraverseRef(),
				},
				// TODO support inclusion and explusion parameters, though, we need to handle the reader
				// role too, not only the argument here.
				// subscription_allow_list= []
				// subscription_deny_list = []
			}),
		)
	)

	blocks, err := tfgen.ObjectsToBlocks(
		providerMondoo,
		providerAzureAD,
		providerAzureRM,
		dataADClientConfig,
		dataRMAllSubscriptions,
		resourceTLSPrivateKey,
		resourceTLSSelfSignedCert,
		resourceADApplicationCertificate,
		resourceADServicePrincipal,
		resourceAdApplication,
		resourceADReadersDirectoryRole,
		resourceRMReaderRoleAssignment,
		resourceADReadersRoleAssignment,
		resourceTimeSleep,
		resourceMondooIntegration,
	)
	if err != nil {
		return "", err
	}

	return tfgen.CreateHclStringOutput(
		tfgen.CombineHclBlocks(requiredProvidersBlock, blocks)...,
	), nil
}

// getOutputDirPath determine output directory location based on how the output location was set
func getOutputDirPath(location string, identifier string) (string, error) {
	// determine code output path
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// If location was passed, return that location
	if location != "" {
		return filepath.FromSlash(location), nil
	}

	// If location was not passed, assemble it with mondoo from os homedir
	// @afiune we should have this location somewhere, look it up
	return filepath.FromSlash(fmt.Sprintf(
		"%s/%s/%s/%s/%s",
		home, ".config", "mondoo", "onboarding", identifier,
	)), nil
}

// WriteHCL Writes the generated HCL code
func WriteHCL(hcl string, location string, identifier string) (string, error) {
	// Determine write location
	dirname, err := getOutputDirPath(location, identifier)
	if err != nil {
		return "", err
	}

	// check if output location exists and if it's a file
	outputDirLocation, err := os.Stat(dirname)
	if !os.IsNotExist(err) && !outputDirLocation.IsDir() {
		return "", fmt.Errorf("output location %s already exists and is a file", dirname)
	}

	// Create directory, if needed
	if os.IsNotExist(err) {
		directory := filepath.FromSlash(dirname)
		if _, err := os.Stat(directory); os.IsNotExist(err) {
			err = os.MkdirAll(directory, 0700)
			if err != nil {
				return "", err
			}
		}
	}

	// Create HCL file
	outputLocation := filepath.FromSlash(fmt.Sprintf("%s/main.tf", dirname))
	err = os.WriteFile(filepath.FromSlash(outputLocation), []byte(hcl), 0700)
	if err != nil {
		return "", err
	}

	return dirname, nil
}

// Execute a terraform plan & apply
func TerraformPlanAndExecute(workingDir, space string) error {
	// Ensure Terraform is installed
	tf, err := LocateOrInstallTerraform(false, workingDir)
	if err != nil {
		return err
	}

	vw := terminal.NewVerboseWriter(10)
	tf.SetStdout(vw)
	tf.SetStderr(vw)

	// Initialize tf project
	if err := TerraformInit(tf); err != nil {
		return err
	}
	vw.Close()

	vw = terminal.NewVerboseWriter(10)
	tf.SetStdout(vw)
	tf.SetStderr(vw)

	// Write plan
	changes, err := TerraformExecPlan(tf)
	if err != nil {
		return err
	}
	vw.Close()

	// Display changes and determine if apply should proceed
	proceed, err := DisplayTerraformPlanChanges(tf, *changes)
	if err != nil {
		return err
	}

	// If not proceed; display guidance on how to continue outside of this session
	if !proceed {
		fmt.Println(provideGuidanceAfterExit(true, tf.WorkingDir(), tf.ExecPath()))
		return nil
	}

	vw = terminal.NewVerboseWriter(10)
	tf.SetStdout(vw)
	tf.SetStderr(vw)

	// Apply plan
	if err := TerraformExecApply(tf); err != nil {
		return errors.New(provideGuidanceAfterFailure(err, tf.WorkingDir(), tf.ExecPath()))
	}
	vw.Close()

	log.Info().Msgf(
		"Mondoo integration was successful! Automation code saved in %s",
		theme.DefaultTheme.Success(tf.WorkingDir()),
	)
	log.Info().Msgf(
		"To view integration status, visit https://console.mondoo.com/space/integrations/azure?spaceId=%s",
		space,
	)
	return nil
}

type terraformVersion struct {
	Version string `json:"terraform_version"`
}

// LocateOrInstallTerraform Determine if terraform is installed, if that version is new enough,
// and if not install a new ephemeral binary of the correct version into tmp location
//
// forceInstall: if set always install ephemeral binary
func LocateOrInstallTerraform(forceInstall bool, workingDir string) (*tfexec.Terraform, error) {
	// find existing binary if not force install
	execPath, _ := exec.LookPath("terraform")
	if execPath != "" {
		log.Info().Msgf("existing Terraform path %s", theme.DefaultTheme.Primary(execPath))
	}

	existingVersionOk := false
	if !forceInstall && execPath != "" {
		// Test if it's an OK version
		requiredVersion := requiredTerraformVersion
		constraint, _ := semver.NewConstraint(requiredVersion)

		// Extract tf version && check for unsupportedExistingVersion
		out, err := exec.Command("terraform", "--version", "--json").Output()
		if err != nil {
			return nil,
				errors.Wrap(err,
					fmt.Sprintf("failed to collect version from existing Terraform install (%s)", execPath),
				)
		}

		// If this version supports checking the version via --version --json, check if we can use it
		var data terraformVersion
		unsupportedVersionCheck := false
		err = json.Unmarshal(out, &data)
		if err != nil {
			// If this version does not support checking version via  --version --json, report and install new
			log.Warn().Msgf(
				"existing Terraform version cannot be used, version doesn't meet requirement %s, "+
					"installing short lived version",
				theme.DefaultTheme.Secondary(requiredVersion),
			)
			unsupportedVersionCheck = true
		}
		log.Info().Msgf("existing Terraform version %s", theme.DefaultTheme.Secondary(data.Version))

		// Parse into new semver
		if !unsupportedVersionCheck {
			tfVersion, err := semver.NewVersion(data.Version)
			if err != nil {
				return nil,
					errors.Wrap(err,
						fmt.Sprintf("version from existing Terraform install is invalid (%s)", data.Version),
					)
			}

			// Test if it matches
			existingVersionOk, _ = constraint.Validate(tfVersion)
			if !existingVersionOk {
				log.Info().Msgf(
					"existing Terraform version cannot be used, version %s doesn't meet requirement %s, "+
						"installing short lived version\n",
					theme.DefaultTheme.Error(data.Version),
					theme.DefaultTheme.Success(requiredVersion),
				)
			}
			log.Info().Msg("using existing Terraform install")
		}
	}

	if !existingVersionOk {
		// If forceInstall was true or the existing version couldn't be used, install it
		installer := &releases.ExactVersion{
			Product: product.Terraform,
			Version: version.Must(version.NewVersion(installTerraformVersion)),
		}

		log.Info().Msgf("installing Terraform version %s", theme.DefaultTheme.Primary(installTerraformVersion))
		installPath, err := installer.Install(context.Background())
		if err != nil {
			return nil, errors.Wrap(err, "error installing terraform")
		}
		execPath = installPath
	}

	return newTf(workingDir, execPath)
}

// helper function to create new *tfexec.Terraform object and return useful error if not found
func newTf(workingDir string, execPath string) (*tfexec.Terraform, error) {
	// Create new tf object
	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not locate terraform binary")
	}

	return tf, nil
}

func TerraformInit(tf *tfexec.Terraform) error {
	log.Info().Msgf("initializing automation code %s", theme.DefaultTheme.Primary("(terraform init)"))
	err := tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		return errors.Wrap(err, "failed to execute terraform init")
	}

	return nil
}

// TerraformExecPlan Run terraform plan using the workingDir from *tfexec.Terraform
//
// - Run plan
// - Get plan file details (returned)
func TerraformExecPlan(tf *tfexec.Terraform) (*TfPlanChangesSummary, error) {
	log.Info().Msgf("creating execution plan %s", theme.DefaultTheme.Primary("(terraform plan)"))
	_, err := tf.Plan(context.Background(), tfexec.Out("tfplan.json"))
	if err != nil {
		return nil, err
	}

	// disale virtual terminal output since the following command overloads the terminal
	tf.SetStdout(io.Discard)

	return processTfPlanChangesSummary(tf)
}

func processTfPlanChangesSummary(tf *tfexec.Terraform) (*TfPlanChangesSummary, error) {
	// Extract changes from tf plan
	log.Info().Msg("getting terraform plan details")
	plan, err := tf.ShowPlanFile(context.Background(), "tfplan.json")
	if err != nil {
		return nil, errors.Wrap(err, "failed to inspect terraform plan")
	}

	return parseTfPlanOutput(plan), nil
}
func parseTfPlanOutput(plan *tfjson.Plan) *TfPlanChangesSummary {
	// Build output of changes
	resourceCreate := 0
	resourceDelete := 0
	resourceUpdate := 0
	resourceReplace := 0

	for _, c := range plan.ResourceChanges {
		switch {
		case c.Change.Actions.Create():
			resourceCreate++
		case c.Change.Actions.Delete():
			resourceDelete++
		case c.Change.Actions.Update():
			resourceUpdate++
		case c.Change.Actions.Replace():
			resourceReplace++
		}
	}

	return &TfPlanChangesSummary{
		plan:    plan,
		create:  resourceCreate,
		deleted: resourceDelete,
		update:  resourceDelete,
		replace: resourceReplace,
	}
}

// TerraformExecApply Run terraform apply using the workingDir from *tfexec.Terraform
//
// - Run plan
// - Get plan file details (returned)
func TerraformExecApply(tf *tfexec.Terraform) error {
	log.Info().Msgf("running automation %s", theme.DefaultTheme.Primary("(terraform apply)"))
	err := tf.Apply(context.Background())
	if err != nil {
		return err
	}

	return nil
}

type TfPlanChangesSummary struct {
	plan    *tfjson.Plan
	create  int
	deleted int
	update  int
	replace int
}

func provideGuidanceAfterFailure(err error, workingDir string, binaryLocation string) string {
	out := new(strings.Builder)
	fmt.Fprintf(out, "\n\n%s\n\n", err.Error())
	fmt.Fprintln(out, strings.Repeat("-", 80))
	fmt.Fprint(out, "Terraform encountered an error (see above)\n\n")
	fmt.Fprintf(out, "The Terraform code, state, and plan output have been saved in %s.\n\n", workingDir)
	fmt.Fprintln(out, "Once the issues have been resolved, the integration can be continued using the following commands:")
	fmt.Fprintf(out, "  cd %s\n", workingDir)
	fmt.Fprintf(out, "  %s apply\n\n", binaryLocation)
	fmt.Fprintln(out, "Should you simply want to clean up the failed deployment, use the following commands:")
	fmt.Fprintf(out, "  cd %s\n", workingDir)
	fmt.Fprintf(out, "  %s destroy\n\n", binaryLocation)

	return out.String()
}

type azAccount struct {
	CloudName        string `json:"cloudName"`
	HomeTenantID     string `json:"homeTenantId"`
	ID               string `json:"id"`
	IsDefault        bool   `json:"isDefault"`
	ManagedByTenants []any  `json:"managedByTenants"`
	Name             string `json:"name"`
	State            string `json:"state"`
	TenantID         string `json:"tenantId"`
	User             struct {
		CloudShellID bool   `json:"cloudShellID"`
		Name         string `json:"name"`
		Type         string `json:"type"`
	} `json:"user"`
}

// Printable Keys and Values are used by the cnquery/cli/components package.
var assetPrintableKeys = []string{"name", "subscription-id"}

func (a azAccount) PrintableKeys() []string {
	return assetPrintableKeys
}
func (a azAccount) PrintableValue(index int) string {
	switch assetPrintableKeys[index] {
	case "name":
		return a.Name
	case "subscription-id":
		if a.IsDefault {
			return fmt.Sprintf("(default) %s", a.ID)
		}
		return a.ID
	default:
		return a.Display()
	}
}

// Display implements SelectableItem from the cnquery/cli/components package.
func (az azAccount) Display() string {
	if az.IsDefault {
		return fmt.Sprintf("(%s) %s [default]", az.ID, az.Name)
	}
	return fmt.Sprintf("(%s) %s", az.ID, az.Name)
}

// DisplayTerraformPlanChanges used to display the results of a plan
//
// returns true if apply should run, false to exit
func DisplayTerraformPlanChanges(tf *tfexec.Terraform, data TfPlanChangesSummary) (bool, error) {
	// Prompt for next steps
	prompt := true
	previewShown := false
	var answer int

	// Displaying resources
	for prompt {
		id := promptForTerraformNextSteps(&previewShown, data)

		switch {
		case id == 1 && !previewShown:
			fmt.Println(buildHumanReadablePlannedActions(tf.WorkingDir(), tf.ExecPath(), data.plan.ResourceChanges))
		default:
			answer = id
			prompt = false
		}

		if id == 1 && !previewShown {
			previewShown = true
		}
	}

	// Run apply
	if answer == 0 {
		return true, nil
	}

	// Quit
	return false, nil
}

// buildHumanReadablePlannedActions creates a summarized listing of expected changes from Terraform
func buildHumanReadablePlannedActions(workingDir, execPath string, data []*tfjson.ResourceChange) string {
	outputString := strings.Builder{}
	outputString.WriteString("Resource details:\n")

	for _, c := range data {
		outputString.WriteString(fmt.Sprintf("  %s.%s will be %s\n", c.Type, c.Name,
			createOrDestroy(
				c.Change.Actions.Create(),
				c.Change.Actions.Delete(),
				c.Change.Actions.Update(),
				c.Change.Actions.Read(),
				c.Change.Actions.NoOp(),
				c.Change.Actions.Replace(),
				c.Change.Actions.CreateBeforeDestroy(),
				c.Change.Actions.DestroyBeforeCreate(),
			),
		),
		)
	}
	outputString.WriteString("\n")
	outputString.WriteString(fmt.Sprintf(
		"More details can be viewed by running:\n\n  cd %s\n  %s show tfplan.json",
		workingDir, execPath,
	))
	outputString.WriteString("\n")
	return outputString.String()
}

// used to create suitable response messages when showing actions tf will take as a result of a plan execution
func createOrDestroy(create, destroy, update, read, noop, replace, createBfDestroy, destroyBfCreate bool) string {
	switch {
	case create:
		return theme.DefaultTheme.Success("created")
	case destroy:
		return theme.DefaultTheme.Error("destroyed")
	case update:
		return theme.DefaultTheme.Primary("update")
	case read:
		return theme.DefaultTheme.Secondary("read")
	case replace:
		return theme.DefaultTheme.Primary("replaced")
	case createBfDestroy:
		return theme.DefaultTheme.Success("created before destroy")
	case destroyBfCreate:
		return theme.DefaultTheme.Error("destroyed before create")
	case noop:
		return "(noop)"
	default:
		return "unchanged"
	}
}

type simpleOption string

func (s simpleOption) Display() string {
	return string(s)
}

// Simple helper to prompt for next steps after TF plan
func promptForTerraformNextSteps(previewShown *bool, data TfPlanChangesSummary) int {
	options := []simpleOption{
		"Continue to apply changes",
	}

	// Omit option to show details if we already have
	if !*previewShown {
		options = append(options, "Show details")
	}
	options = append(options, "Quit")

	return components.Select(
		fmt.Sprintf(
			"The automation will create %d resources, delete %d resources, update %d resources, and replace %d resources.",
			data.create, data.deleted, data.update, data.replace,
		),
		options,
	)
}

// this helper function is called when the entire generation/apply flow is not completed; it provides
// guidance on how to proceed from the last point of execution
func provideGuidanceAfterExit(initRun bool, workingDir string, binaryLocation string) string {
	out := new(strings.Builder)
	fmt.Fprintf(out, "Automation code and plan output saved in %s\n\n", theme.DefaultTheme.Secondary(workingDir))
	fmt.Fprintf(out, "The generated code can be executed at any point in the future using the following commands:\n")
	fmt.Fprintf(out, "  cd %s\n", workingDir)

	if !initRun {
		fmt.Fprintf(out, "  %s init\n", binaryLocation)
	}

	fmt.Fprintf(out, "  %s plan\n", binaryLocation)
	fmt.Fprintf(out, "  %s apply\n\n", binaryLocation)
	return out.String()
}

func generateIntegrationName(subscription string) string {
	var subsSplit = strings.Split(subscription, "-")
	return "subscription-" + subsSplit[len(subsSplit)-1]
}
