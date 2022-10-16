package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/muesli/termenv"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/cli/theme"
	"go.mondoo.com/cnquery/cli/theme/colors"
	"go.mondoo.com/cnquery/logger"
)

const (
	rootCmdDesc = "cnspec is a cloud-native security testing tool for your entire fleet\n"
)

const cnspecLogo = "  ___ _ __  ___ _ __   ___  ___ \n" +
	" / __| '_ \\/ __| '_ \\ / _ \\/ __|\n" +
	"| (__| | | \\__ \\ |_) |  __/ (__ \n" +
	" \\___|_| |_|___/ .__/ \\___|\\___|\n" +
	"   mondoo™     |_|              "

func init() {
	theme.DefaultTheme.Landing = landing()
	theme.DefaultTheme.Welcome = welcome()
	theme.DefaultTheme.Prefix = "cnspec> "
}

func landing() string {
	// windows
	if runtime.GOOS == "windows" {
		return termenv.String(cnspecLogo + "\n").Foreground(colors.DefaultColorTheme.Primary).String()
	}
	// unix
	return termenv.String(cnspecLogo).Foreground(colors.DefaultColorTheme.Primary).String()
}

func welcome() string {
	// windows
	if runtime.GOOS == "windows" {
		return cnspecLogo + " interactive shell\n"
	}
	// unix
	return cnspecLogo + " interactive shell\n"
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cnspec",
	Short: "cnspec CLI",
	// NOTE: if we use theme.DefaultTheme.Landing go compiler uses the value before init updated it
	Long: landing() + "\n\n" + rootCmdDesc,
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

func GenerateMarkdown(dir string) error {
	rootCmd.DisableAutoGenTag = true

	// We need to remove our fancy logo from the markdown output,
	// since it messes with the formatting.
	rootCmd.Long = rootCmdDesc
	return doc.GenMarkdownTree(rootCmd, dir)
}
