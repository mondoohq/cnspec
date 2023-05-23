package plugin

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/muesli/termenv"
	zlog "github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/shared"
	"go.mondoo.com/cnquery/shared/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func addColorConfig(cmd *exec.Cmd) {
	switch termenv.EnvColorProfile() {
	case termenv.ANSI256, termenv.ANSI, termenv.TrueColor:
		cmd.Env = append(cmd.Env, "CLICOLOR_FORCE=1")
	default:
		// it will default to no-color, since it's run as a plugin
		// so there is nothing to do here
	}
}

func cnqueryLocation() string {
	if e := os.Getenv("CNQUERY_PLUGIN"); e != "" {
		return e
	}

	// the default is to use the available cnquery command
	return "cnquery"
}

func RunQuery(conf *proto.RunQueryConfig) error {
	// disable the plugin's logs
	pluginLogger := hclog.New(&hclog.LoggerOptions{
		Name: "cnquery-plugin",
		// Level: hclog.LevelFromString("DEBUG"),
		Level:  hclog.Info,
		Output: ioutil.Discard,
	})

	pluginCmd := exec.Command(cnqueryLocation(), "run_as_plugin")
	zlog.Debug().Msgf("running cnquery from: '%s'", pluginCmd.Path)

	addColorConfig(pluginCmd)

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: shared.Handshake,
		Plugins:         shared.PluginMap,
		Cmd:             pluginCmd,
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC, plugin.ProtocolGRPC,
		},
		Logger: pluginLogger,
		Stderr: os.Stderr,
	})
	defer client.Kill()

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		return errors.Wrap(err, "failed to initialize plugin client")
	}

	// Request the plugin
	pluginName := "cnquery"
	raw, err := rpcClient.Dispense(pluginName)
	if err != nil {
		return errors.Wrap(err, "failed to call "+pluginName+" plugin")
	}

	cnquery := raw.(shared.CNQuery)

	writer := shared.IOWriter{Writer: os.Stdout}
	err = cnquery.RunQuery(conf, &writer)
	if err != nil {
		if status, ok := status.FromError(err); ok {
			code := status.Code()
			switch code {
			case codes.Unavailable, codes.Internal:
				return errors.New(pluginName + " plugin crashed, please report any stack trace you see with this error")
			case codes.Unimplemented:
				return errors.New(pluginName + " plugin failed, the call is not implemented, please report this error")
			default:
				return errors.New(pluginName + " plugin failed, error " + strconv.Itoa(int(code)) + ": " + status.Message())
			}
		}

		return err
	}

	return nil
}
