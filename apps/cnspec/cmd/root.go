package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/cli/components"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/cli/theme"
	"go.mondoo.com/cnquery/logger"
)

const (
	rootCmdDesc = "cnspec is a cloud-native testing tool for your entire fleet\n"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cnspec",
	Short: "cnspec CLI",
	Long:  theme.DefaultTheme.Landing + "\n\n" + rootCmdDesc,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initLogger(cmd)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// normal cli handling
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// NOTE: we need to call this super early, otherwise the CLI color output on Windows is broken for the first lines
	// since the log instance is already initialized, replace default zerolog color output with our own
	// use color logger by default
	logger.CliCompactLogger(logger.LogOutputWriter)
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	config.DefaultConfigFile = "mondoo.yml"

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().String("log-level", "info", "set log-level: error, warn, info, debug, trace")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindEnv("features")

	config.Init(rootCmd)
}

func initLogger(cmd *cobra.Command) {
	// environment variables always over-write custom flags
	envLevel, ok := logger.GetEnvLogLevel()
	if ok {
		logger.Set(envLevel)
		return
	}

	// retrieve log-level from flags
	level := viper.GetString("log-level")
	if v := viper.GetBool("verbose"); v {
		level = "debug"
	}
	logger.Set(level)
}

func askForPassword(prompt string, flagset *flag.FlagSet) {
	// check if password is set
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		log.Fatal().Msg("--ask-pass is only supported when used with a TTY")
	}

	// ask user for password
	passwordModel := components.NewPasswordModel(prompt, func(userPassword string, aborted bool) {
		flagset.Set("password", userPassword)
		if aborted {
			os.Exit(1)
		}
	})
	p := tea.NewProgram(passwordModel)
	if err := p.Start(); err != nil {
		panic(err)
	}
}
