// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	tea "github.com/charmbracelet/bubbletea"
	"go.mondoo.com/mql/providers"
)

// A connector's input form is built from metadata that only an installed
// provider carries: providers.DefaultProviders names every connector cnspec
// supports but strips the flags, arg counts and discovery targets. So the
// launcher installs a provider on demand the moment a user opens its form,
// rather than degrading to a bare text box for anything not already on disk.
//
// The install is the same one a scan would trigger on first use, so this only
// moves the download earlier -- to the point where the user is choosing what to
// run rather than in the middle of running it.

// providerInstalledMsg reports the outcome of an on-demand install.
type providerInstalledMsg struct {
	provider string
	// conns are the provider's connectors rebuilt from the freshly loaded
	// metadata, ready to replace the catalog's placeholder entries.
	conns []Connector
	err   error
}

// installProviderCmd downloads and installs a provider in the background.
//
// EnsureProvider does network I/O and can take seconds, so this must only ever
// run as a tea.Cmd. It also writes the package-level provider cache without a
// lock, so the caller is responsible for keeping a single install in flight.
func installProviderCmd(provider string) tea.Cmd {
	return func() tea.Msg {
		p, err := providers.EnsureProvider(
			providers.ProviderLookup{ProviderName: provider}, true, nil)
		if err != nil {
			return providerInstalledMsg{provider: provider, err: err}
		}
		// EnsureProvider loads the provider's JSON on install, so p carries the
		// full connector metadata the form needs.
		return providerInstalledMsg{
			provider: provider,
			conns:    connectorsOf(provider, p.Provider, true),
		}
	}
}

// What the launcher does with the answer is listState.applyInstalled: the
// catalog is what an install replaces, and the catalog is the list's.
