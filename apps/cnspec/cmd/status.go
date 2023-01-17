package cmd

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery"
	cnquery_cmd "go.mondoo.com/cnquery/apps/cnquery/cmd"
	cnquery_config "go.mondoo.com/cnquery/apps/cnquery/cmd/config"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/cli/sysinfo"
	"go.mondoo.com/cnquery/cli/theme"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/local"
	"go.mondoo.com/cnquery/upstream"
	"go.mondoo.com/cnquery/upstream/health"
	"go.mondoo.com/ranger-rpc"
	"sigs.k8s.io/yaml"
)

func init() {
	statusCmd.Flags().StringP("output", "o", "", "Set output format. Accepts json or yaml")
	rootCmd.AddCommand(statusCmd)
}

// statusCmd represents the version command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Verify access to Mondoo Platform.",
	Long: `
Status sends a ping to Mondoo Platform to verify the credentials.
	`,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("output", cmd.Flags().Lookup("output"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		opts, optsErr := cnquery_config.ReadConfig()
		if optsErr != nil {
			log.Fatal().Err(optsErr).Msg("could not load configuration")
		}

		config.DisplayUsedConfig()

		s := Status{
			Client: ClientStatus{
				Timestamp: time.Now().Format(time.RFC3339),
				Version:   cnquery.GetVersion(),
				Build:     cnquery.GetBuild(),
			},
		}

		provider, err := local.New()
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		m, err := motor.New(provider)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		sysInfo, err := sysinfo.GatherSystemInfo(sysinfo.WithMotor(m))
		if err == nil {
			s.Client.Platform = sysInfo.Platform
			s.Client.Hostname = sysInfo.Hostname
			s.Client.IP = sysInfo.IP
		}

		// check server health and clock skew
		upstreamStatus, err := health.CheckApiHealth(opts.UpstreamApiEndpoint())
		if err != nil {
			log.Error().Err(err).Msg("could not check upstream health")
		}
		s.Upstream = upstreamStatus

		// check valid agent authentication
		plugins := []ranger.ClientPlugin{}
		// plugins = append(plugins, max.NewClientInfoPlugin(clientInfo, opts.GetFeatures()))

		// try to load config into credentials struct
		credentials := opts.GetServiceCredential()
		if credentials != nil && len(credentials.Mrn) > 0 {
			s.Client.ParentMrn = credentials.ParentMrn
			s.Client.Registered = true
			s.Client.ServiceAccount = credentials.Mrn
			s.Client.Mrn = opts.AgentMrn
			if s.Client.Mrn == "" {
				s.Client.Mrn = "no managed client"
			}

			certAuth, err := upstream.NewServiceAccountRangerPlugin(credentials)
			if err != nil {
				log.Error().Err(err).Msg(errorMessageServiceAccount)
				os.Exit(cnquery_cmd.ConfigurationErrorCode)
			}
			plugins = append(plugins, certAuth)

			// try to ping the server
			client, err := upstream.NewAgentManagerClient(s.Upstream.API.Endpoint, ranger.DefaultHttpClient(), plugins...)
			if err == nil {
				_, err = client.PingPong(context.Background(), &upstream.Ping{})
				if err != nil {
					s.Client.PingPongError = err
				}
			} else {
				s.Client.PingPongError = err
			}
		}

		switch strings.ToLower(viper.GetString("output")) {
		case "yaml":
			s.RenderYaml()
		case "json":
			s.RenderJson()
		default:
			s.RenderCliStatus()
		}

		if !s.Client.Registered || s.Client.PingPongError != nil {
			os.Exit(1)
		}
	},
}

type Status struct {
	Client   ClientStatus  `json:"client"`
	Upstream health.Status `json:"upstream"`
}

type ClientStatus struct {
	Timestamp      string             `json:"timestamp,omitempty"`
	Mrn            string             `json:"mrn,omitempty"`
	ServiceAccount string             `json:"service_account,omitempty"`
	ParentMrn      string             `json:"parentMrn,omitempty"`
	Version        string             `json:"version,omitempty"`
	Build          string             `json:"build,omitempty"`
	Labels         map[string]string  `json:"labels,omitempty"`
	Platform       *platform.Platform `json:"platform,omitempty"`
	IP             string             `json:"ip,omitempty"`
	Hostname       string             `json:"hostname,omitempty"`
	Registered     bool               `json:"registered,omitempty"`
	PingPongError  error              `json:"pingPongError,omitempty"`
}

func (s Status) RenderCliStatus() {
	if s.Client.Platform != nil {
		agent := s.Client
		log.Info().Msg("Platform:\t" + agent.Platform.Name)
		log.Info().Msg("Version:\t" + agent.Platform.Version)
		log.Info().Msg("Hostname:\t" + agent.Hostname)
		log.Info().Msg("IP:\t\t" + agent.IP)
	} else {
		log.Warn().Msg("could not determine client platform information")
	}

	log.Info().Msg("Time:\t\t" + s.Client.Timestamp)
	log.Info().Msg("Version:\t" + cnquery.GetVersion() + " (API Version: " + cnquery.APIVersion() + ")")

	log.Info().Msg("API ConnectionConfig:\t" + s.Upstream.API.Endpoint)
	log.Info().Msg("API Status:\t" + s.Upstream.API.Status)
	log.Info().Msg("API Time:\t" + s.Upstream.API.Timestamp)
	log.Info().Msg("API Version:\t" + s.Upstream.API.Version)

	if s.Upstream.API.Version != cnquery.APIVersion() {
		log.Warn().Msg("API versions do not match, please update the client")
	}

	if len(s.Upstream.Features) > 0 {
		log.Info().Msg("Features:\t" + strings.Join(s.Upstream.Features, ","))
	}
	log.Info().Msg("Owner:\t" + s.Client.ParentMrn)

	if s.Client.Registered {
		log.Info().Msg("Client:\t" + s.Client.Mrn)
		log.Info().Msg("Service Account:\t" + s.Client.ServiceAccount)
		log.Info().Msg(theme.DefaultTheme.Success("client is registered"))
	} else {
		log.Error().Msg("client is not registered")
	}

	if s.Client.Registered && s.Client.PingPongError == nil {
		log.Info().Msg(theme.DefaultTheme.Success("client authenticated successfully"))
	} else {
		log.Error().Err(s.Client.PingPongError).Msg("could not connect to mondoo platform")
	}

	for i := range s.Upstream.Warnings {
		log.Warn().Msg(s.Upstream.Warnings[i])
	}
}

func (s Status) RenderJson() {
	output, err := json.Marshal(s)
	if err != nil {
		log.Error().Err(err).Msg("could not generate json")
	}
	os.Stdout.Write(output)
}

func (s Status) RenderYaml() {
	output, err := yaml.Marshal(s)
	if err != nil {
		log.Error().Err(err).Msg("could not generate yaml")
	}
	os.Stdout.Write(output)
}
