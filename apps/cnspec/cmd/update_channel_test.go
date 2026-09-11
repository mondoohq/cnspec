// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/cli/config"
)

func updateFlagCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "update"}
	cmd.Flags().String("channel", "", "")
	require.NoError(t, cmd.ParseFlags(args))
	return cmd
}

func resetUpdateChannelState(t *testing.T) {
	t.Helper()
	viper.Set(config.KeyUpdateChannel, "")
	config.SetRunningVersion("")
	t.Cleanup(func() {
		viper.Set(config.KeyUpdateChannel, "")
		config.SetRunningVersion("")
	})
}

// TestResolveUpdateChannel covers the one-off override on `cnspec update`:
// moving a stable install to a release candidate without putting the machine on
// the pre-release track permanently.
func TestResolveUpdateChannel(t *testing.T) {
	t.Run("no flag uses the configured channel", func(t *testing.T) {
		resetUpdateChannelState(t)
		viper.Set(config.KeyUpdateChannel, config.ChannelPreview)

		channel, err := resolveUpdateChannel(updateFlagCmd(t))
		require.NoError(t, err)
		assert.Equal(t, config.ChannelPreview, channel)
	})

	t.Run("no flag on a stable build is stable", func(t *testing.T) {
		resetUpdateChannelState(t)
		config.SetRunningVersion("13.38.1")

		channel, err := resolveUpdateChannel(updateFlagCmd(t))
		require.NoError(t, err)
		assert.Equal(t, config.ChannelStable, channel)
	})

	t.Run("the flag overrides a stable build", func(t *testing.T) {
		resetUpdateChannelState(t)
		config.SetRunningVersion("13.38.1")

		channel, err := resolveUpdateChannel(updateFlagCmd(t, "--channel", "preview"))
		require.NoError(t, err)
		assert.Equal(t, config.ChannelPreview, channel)
	})

	t.Run("the flag overrides configuration", func(t *testing.T) {
		resetUpdateChannelState(t)
		viper.Set(config.KeyUpdateChannel, config.ChannelPreview)

		channel, err := resolveUpdateChannel(updateFlagCmd(t, "--channel", "stable"))
		require.NoError(t, err)
		assert.Equal(t, config.ChannelStable, channel)
	})

	t.Run("case and spacing are normalized", func(t *testing.T) {
		resetUpdateChannelState(t)

		channel, err := resolveUpdateChannel(updateFlagCmd(t, "--channel", "  PREVIEW "))
		require.NoError(t, err)
		assert.Equal(t, config.ChannelPreview, channel)
	})

	t.Run("an unknown channel is an error, not a fallback", func(t *testing.T) {
		resetUpdateChannelState(t)

		_, err := resolveUpdateChannel(updateFlagCmd(t, "--channel", "beta"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "beta")
		assert.Contains(t, err.Error(), config.ChannelPreview)
	})
}

// TestIsPrereleaseVersion pins the check behind the "cnspec does not downgrade"
// message, including the shapes a development build reports.
func TestIsPrereleaseVersion(t *testing.T) {
	for version, want := range map[string]bool{
		"14.0.0-rc.2":      true,
		"14.0.0-rc.2.3509": true,
		"v14.0.0-pre.1":    true,
		"14.0.0-rolling":   true,
		"13.38.1":          false,
		"14.0.0":           false,
		"8.4.0+41":         false, // build metadata is not a pre-release
		"unstable":         false,
		"":                 false,
	} {
		assert.Equal(t, want, isPrereleaseVersion(version), "version: %q", version)
	}
}
