package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cnquery_cmd "go.mondoo.com/cnquery/apps/cnquery/cmd"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/cli/sysinfo"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/upstream"
	cnspec_config "go.mondoo.com/cnspec/apps/cnspec/cmd/config"
	"go.mondoo.com/cnspec/policy/scan"
	"go.mondoo.com/ranger-rpc"
)

func init() {
	serveApiCmd.Flags().String("address", "127.0.0.1", "address to listen on")
	serveApiCmd.Flags().Uint("port", 8080, "port to listen on")
	rootCmd.AddCommand(serveApiCmd)
}

var serveApiCmd = &cobra.Command{
	Use:    "serve-api",
	Hidden: true,
	Short:  "EXPERIMENTAL: Serve a REST API for running scans",
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("port", cmd.Flags().Lookup("port"))
		viper.BindPFlag("address", cmd.Flags().Lookup("address"))

		logger.StandardZerologLogger()

		// TODO: will be added later
		// viper.BindPFlag("token", cmd.Flags().Lookup("token"))
		// viper.BindPFlag("token-file-path", cmd.Flags().Lookup("token-file-path"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Warn().Msg("this is an experimental feature, use at your own risk")
		opts, optsErr := cnspec_config.ReadConfig()
		if optsErr != nil {
			log.Fatal().Err(optsErr).Msg("could not load configuration")
		}
		config.DisplayUsedConfig()

		serviceAccount := opts.GetServiceCredential()
		if serviceAccount == nil {
			log.Fatal().Msg("no service account configured")
		}

		certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
		if err != nil {
			log.Error().Err(err).Msg("could not initialize client authentication")
			os.Exit(cnquery_cmd.ConfigurationErrorCode)
		}
		plugins := []ranger.ClientPlugin{certAuth}
		// determine information about the client
		sysInfo, err := sysinfo.GatherSystemInfo()
		if err != nil {
			log.Warn().Err(err).Msg("could not gather client information")
		}
		plugins = append(plugins, defaultRangerPlugins(sysInfo, opts.GetFeatures())...)
		log.Info().Msg("using service account credentials")
		upstreamConfig := resources.UpstreamConfig{
			SpaceMrn:    opts.GetParentMrn(),
			ApiEndpoint: opts.UpstreamApiEndpoint(),
			Plugins:     plugins,
		}

		scanner := scan.NewLocalScanner(scan.WithUpstream(upstreamConfig.ApiEndpoint, upstreamConfig.SpaceMrn), scan.WithPlugins(plugins), scan.DisableProgressBar())
		if err := scanner.EnableQueue(); err != nil {
			log.Fatal().Err(err).Msg("could not enable scan queue")
		}

		addressOpt := viper.GetString("address")
		portOpt := viper.GetInt("port")
		bind, err := getHttpBind(addressOpt, portOpt)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create HTTP bind")
		}

		uri, err := url.Parse(bind)
		if err != nil {
			log.Fatal().Err(err).Str("binding", bind).Msg("failed to parse binding")
		}

		mux := http.NewServeMux()
		server := scan.NewScanServer(scanner)
		log.Info().Str("url", "/Scan/").Msg("enable Scanner API")
		mux.Handle("/Scan/", server)

		if err := bindHTTP(mux, uri); err != nil {
			log.Fatal().Err(err).Msg("failed to bind http server")
		}
	},
}

func bindHTTP(mux http.Handler, uri *url.URL) error {
	addr := uri.Host
	log.Info().Str("address", addr).Msg("start http server")

	server := http.Server{
		Handler: mux,
	}

	var tcpListener net.Listener
	var err error

	// http listener
	tcpListener, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, os.Interrupt)

	// graceful shutdown function
	go func() {
		<-quit
		log.Info().Msg("shutting down server")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("could not gracefully shutdown the server")
		}
		close(done)
	}()

	err = server.Serve(tcpListener)
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	<-done
	log.Info().Msg("shutdown server successfully")
	return nil
}

func getHttpBind(address string, port int) (string, error) {
	// For now support only http
	return fmt.Sprintf("%s://%s:%s", "http", address, strconv.Itoa(port)), nil
}
