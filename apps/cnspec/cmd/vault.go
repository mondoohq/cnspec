package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnquery/motor/vault/config"
)

func init() {
	vaultListCmd.Flags().Bool("show-options", false, "displays configured options")
	vaultCmd.AddCommand(vaultListCmd)

	vaultConfigureCmd.Flags().String("type", "", "possible values: "+strings.Join(vault.TypeIds(), " | "))
	vaultConfigureCmd.Flags().StringToString("option", nil, "addition vault connection options, multiple options via --option key=value")
	vaultCmd.AddCommand(vaultConfigureCmd)

	vaultCmd.AddCommand(vaultRemoveCmd)
	vaultCmd.AddCommand(vaultResetCmd)

	vaultCmd.AddCommand(vaultAddSecretCmd)

	rootCmd.AddCommand(vaultCmd)
}

func emptyVaultConfigSecret() *vault.Secret {
	return &vault.Secret{
		Key:   config.VaultConfigStoreKey,
		Label: "User Vault Settings",
		Data:  config.ClientVaultConfig{}.SecretData(),
	}
}

// vaultCmd represents the vault command
var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Manage vault environments.",
	Long:  ``,
}

var vaultListCmd = &cobra.Command{
	Use:   "list",
	Short: "List vault environments.",
	Long:  ``,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("show-options", cmd.Flags().Lookup("show-options"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		v := config.GetInternalVault()
		ctx := context.Background()
		secret, err := v.Get(ctx, &vault.SecretID{
			Key: config.VaultConfigStoreKey,
		})
		if err != nil {
			log.Fatal().Msg("no vault configured")
		}

		showOptions := viper.GetBool("show-options")

		vCfgs, err := config.NewClientVaultConfig(secret)
		if err != nil {
			log.Fatal().Err(err).Msg("could not unmarshal credential")
		}

		for k, vCfg := range vCfgs {
			// print configured vault
			fmt.Printf("vault  : %s (%s)\n", k, vCfg.Type.Value())
			// print options if requested
			if showOptions {
				fmt.Printf("options:\n")
				for ko, vo := range vCfg.Options {
					fmt.Printf("  %s = %s\n", ko, vo)
				}
			}
		}
	},
}

var vaultConfigureCmd = &cobra.Command{
	Use:     "configure VAULTNAME",
	Aliases: []string{"set"},
	Short:   "Configure a vault environment.",
	Long: `

cnquery vault set mondoo-client-vault --type linux-kernel-keyring

`,
	Args: cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("type", cmd.Flags().Lookup("type"))
		viper.BindPFlag("option", cmd.Flags().Lookup("option"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		v := config.GetInternalVault()
		ctx := context.Background()

		secret, err := v.Get(ctx, &vault.SecretID{
			Key: config.VaultConfigStoreKey,
		})
		// error happens on initial use, create a new configuration
		if err != nil {
			secret = emptyVaultConfigSecret()
		}

		vCfgs, err := config.NewClientVaultConfig(secret)
		if err != nil {
			log.Fatal().Err(err).Msg("could not load vault configuration")
		}

		// overwrite existing / set vault config
		// field name = vault name
		vt, err := vault.NewVaultType(viper.GetString("type"))
		if err != nil {
			log.Fatal().Err(err).Msg("could not load vault configuration")
		}

		vaultName := args[0]
		cfg := vault.VaultConfiguration{
			Name:    vaultName,
			Type:    vt,
			Options: viper.GetStringMapString("option"),
		}

		vCfgs.Set(vaultName, cfg)
		secret.Data = vCfgs.SecretData()

		log.Info().Str("name", vaultName).Msg("set new vault configuration")
		_, err = v.Set(ctx, secret)
		if err != nil {
			log.Fatal().Err(err).Msg("could not store update into vault")
		}

		log.Info().Msg("stored vault configuration successfully")
	},
}

var vaultRemoveCmd = &cobra.Command{
	Use:   "remove VAULTNAME",
	Short: "Remove a configured vault environment.",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		v := config.GetInternalVault()
		ctx := context.Background()

		secret, err := v.Get(ctx, &vault.SecretID{
			Key: config.VaultConfigStoreKey,
		})
		if err != nil {
			log.Fatal().Err(err).Msg("could not retrieve vault configuration")
		}

		vCfgs, err := config.NewClientVaultConfig(secret)
		if err != nil {
			log.Fatal().Err(err).Msg("could not load vault configuration")
		}

		vaultName := args[0]
		vCfgs.Delete(vaultName)
		secret.Data = vCfgs.SecretData()

		log.Info().Str("name", vaultName).Msg("set new vault configuration")
		_, err = v.Set(ctx, secret)
		if err != nil {
			log.Fatal().Err(err).Msg("could not update vault configuration")
		}

		log.Info().Msg("removed vault configuration successfully")
	},
}

var vaultResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the vault configuration to defaults.",
	Long:  ``,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		v := config.GetInternalVault()
		ctx := context.Background()

		_, err := v.Set(ctx, emptyVaultConfigSecret())
		if err != nil {
			log.Fatal().Err(err).Msg("could not retrieve vault configuration")
		}

		log.Info().Msg("removed vault configuration successfully")
	},
}

var vaultAddSecretCmd = &cobra.Command{
	Use:   "add-secret VAULTNAME SECRETID SECRETVALUE",
	Short: "Store a secret in a vault.",
	Args:  cobra.ExactArgs(3),
	PreRun: func(cmd *cobra.Command, args []string) {
	},
	Run: func(cmd *cobra.Command, args []string) {
		v := config.GetInternalVault()
		ctx := context.Background()

		secret, err := v.Get(ctx, &vault.SecretID{
			Key: config.VaultConfigStoreKey,
		})
		// error happens on initial use, create a new configuration
		if err != nil {
			secret = emptyVaultConfigSecret()
		}

		vCfgs, err := config.NewClientVaultConfig(secret)
		if err != nil {
			log.Fatal().Err(err).Msg("could not load vault configuration")
		}

		// search for vault
		var selectedVaultCfg *vault.VaultConfiguration
		for k, vCfg := range vCfgs {
			if k != args[0] {
				continue
			}
			selectedVaultCfg = &vCfg
		}
		if selectedVaultCfg == nil {
			log.Fatal().Str("vault", args[0]).Msg("could not find vault")
		}

		selectedVault, err := config.New(selectedVaultCfg)
		if err != nil {
			log.Fatal().Msg("could not open vault")
		}

		_, err = selectedVault.Set(ctx, &vault.Secret{
			Key:  args[1],
			Data: []byte(args[2]),
		})
		if err != nil {
			log.Fatal().Msg("could not store secret")
		}
		log.Info().Msg("stored secret successfully")
	},
}
