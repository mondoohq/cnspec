// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

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
	"go.mondoo.com/cnquery/v10/cli/config"
	"go.mondoo.com/cnquery/v10/logger"
	"go.mondoo.com/cnquery/v10/providers"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	cnspec_config "go.mondoo.com/cnspec/v10/apps/cnspec/cmd/config"
	"go.mondoo.com/cnspec/v10/policy/scan"
)

func init() {
	serveApiCmd.Flags().String("address", "127.0.0.1", "address to listen on")
	serveApiCmd.Flags().Uint("port", 8080, "port to listen on")
	serveApiCmd.Flags().Uint("http-timeout", 30, "timeout for http requests in seconds")
	serveApiCmd.Flags().MarkHidden("http-timeout")
	rootCmd.AddCommand(serveApiCmd)
}

var serveApiCmd = &cobra.Command{
	Use:    "serve-api",
	Hidden: true,
	Short:  "EXPERIMENTAL: Serve a REST API for running scans.",
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("port", cmd.Flags().Lookup("port"))
		viper.BindPFlag("address", cmd.Flags().Lookup("address"))
		viper.BindPFlag("http-timeout", cmd.Flags().Lookup("http-timeout"))

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

		log.Info().Msg("using service account credentials")
		upstreamConfig := upstream.UpstreamConfig{
			SpaceMrn:    opts.GetParentMrn(),
			ApiEndpoint: opts.UpstreamApiEndpoint(),
			ApiProxy:    opts.APIProxy,
			Incognito:   false, // because we serve, we interact with upstream, never incognito
			Creds:       serviceAccount,
		}

		scanner := scan.NewLocalScanner(
			scan.WithUpstream(&upstreamConfig),
			scan.DisableProgressBar(),
			scan.WithRecording(providers.NullRecording{}),
		)
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
