package plugin

import (
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/shared"
	"go.mondoo.com/cnquery/shared/proto"
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

	pluginCmd := exec.Command("sh", "-c", cnqueryLocation()+" run_as_plugin")

	addColorConfig(pluginCmd)

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: shared.Handshake,
		Plugins:         shared.PluginMap,
		Cmd:             pluginCmd,
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC, plugin.ProtocolGRPC},
		Logger: pluginLogger,
	})
	defer client.Kill()

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		return err
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("cnquery")
	if err != nil {
		return err
	}

	cnquery := raw.(shared.CNQuery)

	writer := shared.IOWriter{Writer: os.Stdout}
	err = cnquery.RunQuery(conf, &writer)
	if err != nil {
		return err
	}

	return nil
}
