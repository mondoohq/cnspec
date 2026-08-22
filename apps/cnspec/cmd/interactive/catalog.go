// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package interactive

import (
	"go.mondoo.com/cnspec/cli/launcher/catalog"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// The catalog itself now lives in cli/launcher/catalog. What stays here is the
// launcher's spelling of it: `Connector`, the category names, and the two
// entry points. Without these the extraction would have been a rename of every
// line in this package that mentions a connector or a category -- hundreds of
// sites, none of which was the point of moving the file.
//
// Connector is an alias rather than a defined type so that a value crossing
// into cli/launcher/catalog, or arriving from it, needs no conversion and stays
// the same type on both sides.

type Connector = catalog.Connector

const (
	catHosts     = catalog.CatHosts
	catContainer = catalog.CatContainer
	catCloud     = catalog.CatCloud
	catIaC       = catalog.CatIaC
	catIdentity  = catalog.CatIdentity
	catSaaS      = catalog.CatSaaS
	catNetwork   = catalog.CatNetwork
	catDatabase  = catalog.CatDatabase
	catAI        = catalog.CatAI
	catDev       = catalog.CatDev
	catOther     = catalog.CatOther
)

var (
	categoryOrder     = catalog.CategoryOrder
	excludedProviders = catalog.ExcludedProviders
)

func BuildCatalog() []Connector { return catalog.BuildCatalog() }

func connectorsOf(name string, p *plugin.Provider, installed bool) []Connector {
	return catalog.ConnectorsOf(name, p, installed)
}

func categorize(provider, connector string) string {
	return catalog.Categorize(provider, connector)
}
