// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/muesli/termenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v11"
	cnquery_app "go.mondoo.com/cnquery/v11/apps/cnquery/cmd"
	"go.mondoo.com/cnquery/v11/cli/config"
	cli_errors "go.mondoo.com/cnquery/v11/cli/errors"
	"go.mondoo.com/cnquery/v11/cli/providers"
	"go.mondoo.com/cnquery/v11/cli/theme"
	"go.mondoo.com/cnquery/v11/cli/theme/colors"
	"go.mondoo.com/cnquery/v11/logger"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/sysinfo"
	"go.mondoo.com/cnspec/v11"
	"go.mondoo.com/ranger-rpc"
	"go.mondoo.com/ranger-rpc/plugins/scope"
)

const (
	rootCmdDesc = "cnspec is a cloud-native security testing tool for your entire infrastructure\n"
)

const cnspecLogo = "  ___ _ __  ___ _ __   ___  ___ \n" +
	" / __| '_ \\/ __| '_ \\ / _ \\/ __|\n" +
	"| (__| | | \\__ \\ |_) |  __/ (__ \n" +
	" \\___|_| |_|___/ .__/ \\___|\\___|\n" +
	"   mondoo™     |_|              "

const (
	errorMessageServiceAccount = "invalid service account configuration"
)

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

func BuildRootCmd() (*cobra.Command, error) {
	err := providers.AttachCLIs(
		rootCmd,
		&providers.Command{
			Command: shellCmd,
			Run:     shellRun,
			Action:  "Interactive shell with ",
		},
		&providers.Command{
			Command: cnquery_app.RunCmd,
			Run:     cnquery_app.RunCmdRun,
			Action:  "Run a query with ",
		},
		&providers.Command{
			Command: scanCmd,
			Run:     scanCmdRun,
			Action:  "Scan ",
		},
		&providers.Command{
			Command:             vulnCmd,
			Run:                 vulnCmdRun,
			Action:              "Check for vulnerabilities ",
			SupportedConnectors: []string{"docker", "container", "filesystem", "local", "ssh", "vagrant", "winrm", "vsphere", "sbom"},
		},
	)
	return rootCmd, err
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	_, err := BuildRootCmd()
	if err != nil {
		log.Error().Msg(err.Error())
		os.Exit(1)
	}

	// normal cli handling
	if err := rootCmd.Execute(); err != nil {
		if cErr, ok := err.(*cli_errors.CommandError); ok {
			if cErr.HasError() {
				log.Error().Msg(err.Error())
			}
			os.Exit(cErr.ExitCode())
		}

		log.Error().Msg(err.Error())
		os.Exit(1)
	}
}

func init() {
	// NOTE: we need to call this super early, otherwise the CLI color output on Windows is broken for the first lines
	// since the log instance is already initialized, replace default zerolog color output with our own
	// use color logger by default
	logger.CliCompactLogger(logger.LogOutputWriter)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	config.DefaultConfigFile = "mondoo.yml"

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("log-level", "info", "Set log level: error, warn, info, debug, trace")
	rootCmd.PersistentFlags().String("api-proxy", "", "Set proxy for communications with Mondoo API")
	rootCmd.PersistentFlags().Bool("auto-update", true, "Enable automatic provider installation and update")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("api_proxy", rootCmd.PersistentFlags().Lookup("api-proxy"))
	viper.BindPFlag("auto_update", rootCmd.PersistentFlags().Lookup("auto-update"))
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

var reMdName = regexp.MustCompile(`/([^/]+)\.md$`)

func GenerateMarkdown(dir string) error {
	rootCmd.DisableAutoGenTag = true

	// We need to remove our fancy logo from the markdown output,
	// since it messes with the formatting.
	rootCmd.Long = rootCmdDesc

	files := []string{}
	err := doc.GenMarkdownTreeCustom(rootCmd, dir, func(s string) string {
		files = append(files, s)

		titles := reMdName.FindStringSubmatch(s)
		if len(titles) == 0 {
			return ""
		}
		title := strings.ReplaceAll(titles[1], "_", " ")

		return "---\n" +
			"id: " + titles[1] + "\n" +
			"title: " + title + "\n" +
			"---\n\n"
	}, func(s string) string { return s })
	if err != nil {
		return err
	}

	// we need to remove the first headline, since it is doubled with the
	// headline from the ID. Really annoying, all this needs a rewrite.
	for i := range files {
		file := files[i]
		raw, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		if !strings.HasPrefix(string(raw), "---\nid:") {
			continue
		}

		start := strings.Index(string(raw), "\n## ")
		if start < 0 {
			continue
		}

		end := start
		for i := start + 3; i < len(raw); i++ {
			if raw[i] == '\n' {
				end = i
				break
			}
		}

		res := append(raw[0:start], raw[end:]...)
		err = os.WriteFile(file, res, 0o644)
		if err != nil {
			return err
		}
	}

	return nil
}

func defaultRangerPlugins(sysInfo *sysinfo.SystemInfo, features cnquery.Features) []ranger.ClientPlugin {
	plugins := []ranger.ClientPlugin{}
	plugins = append(plugins, scope.NewRequestIDRangerPlugin())
	plugins = append(plugins, sysInfoHeader(sysInfo, features))
	return plugins
}

func sysInfoHeader(sysInfo *sysinfo.SystemInfo, features cnquery.Features) ranger.ClientPlugin {
	const (
		HttpHeaderUserAgent      = "User-Agent"
		HttpHeaderClientFeatures = "Mondoo-Features"
		HttpHeaderPlatformID     = "Mondoo-PlatformID"
	)

	h := http.Header{}
	info := map[string]string{
		"cnspec": cnspec.Version,
		"build":  cnspec.Build,
	}
	if sysInfo != nil {
		info["PN"] = sysInfo.Platform.Name
		info["PR"] = sysInfo.Platform.Version
		info["PA"] = sysInfo.Platform.Arch
		info["IP"] = sysInfo.IP
		info["HN"] = sysInfo.Hostname
		h.Set(HttpHeaderPlatformID, sysInfo.PlatformId)
	}
	h.Set(HttpHeaderUserAgent, scope.XInfoHeader(info))
	h.Set(HttpHeaderClientFeatures, features.Encode())
	return scope.NewCustomHeaderRangerPlugin(h)
}
