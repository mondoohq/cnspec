// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/v11/cli/config"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

const (
	unixUpdateScript    = "https://install.mondoo.com/sh"
	windowsUpdateScript = "https://install.mondoo.com/ps1"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Hidden: true,
	Use:    "update",
	Short:  "Update cnspec to the latest version",
	Long: `This command detects the platform and runs either https://install.mondoo.com/sh
	or https://install.mondoo.com/ps1 to update to the latest package`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// we need silence usage here, otherwise we get the usage printed in case of error
		// see https://github.com/spf13/cobra/issues/340
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true

		opts, optsErr := config.Read()
		if optsErr != nil {
			return errors.New("could not load configuration")
		}

		hc, err := opts.GetHttpClient()
		if err != nil {
			return errors.New("could not create http client")
		}

		return runUpdate(hc)
	},
}

func runUnixUpdate(hc *http.Client) error {
	log.Info().Str("script", unixUpdateScript).Msg("detected linux-based platform, download update script")
	scriptData, err := download(hc, unixUpdateScript)
	if err != nil {
		return err
	}

	// check if bash is available
	_, err = exec.LookPath("bash")
	if err != nil {
		return errors.New("bash is not available, cannot run update script")
	}

	file, err := os.CreateTemp("", "mondoo")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())

	_, err = file.Write(scriptData)
	if err != nil {
		return errors.Wrap(err, "could not write downloaded script")
	}

	log.Info().Str("script", file.Name()).Msg("script downloaded successfully")

	log.Info().Msg("run update script")
	cmd := exec.Command("bash", file.Name())
	return runCmd(cmd)
}

func runWindowsUpdate(hc *http.Client) error {
	log.Info().Str("script", windowsUpdateScript).Msg("detected windows platform, download update script")
	scriptData, err := download(hc, windowsUpdateScript)
	if err != nil {
		return err
	}

	file, err := os.CreateTemp("", "mondoo*.ps1")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())

	_, err = file.Write(scriptData)
	if err != nil {
		return errors.Wrap(err, "could not write downloaded script")
	}
	// we need to close the file, otherwise Powershell cannot read it
	err = file.Close()
	if err != nil {
		return errors.Wrap(err, "could not close downloaded script")
	}

	log.Info().Msg("run update script")
	cmd := exec.Command("powershell", "-c", "Import-module '"+file.Name()+"';Install-Mondoo;")
	return runCmd(cmd)
}

func runUpdate(hc *http.Client) error {
	if runtime.GOOS == "windows" {
		return runWindowsUpdate(hc)
	} else if runtime.GOOS == "darwin" {
		return runUnixUpdate(hc)
	} else if runtime.GOOS == "linux" {
		return runUnixUpdate(hc)
	} else {
		return fmt.Errorf("platform %s is not supported for automatic update", runtime.GOOS)
	}
	return nil
}

func runCmd(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, "update script failed")
	}
	return nil
}

func download(client *http.Client, url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
