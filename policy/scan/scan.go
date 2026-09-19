// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"

	"go.mondoo.com/cnspec/cli/progress"
	"go.mondoo.com/cnspec/policy"
	"go.mondoo.com/mql/providers"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/upstream"
)

// Why do we want the scan stuff as a proto? Because we have a disk queue
// and it makes it easier and faster to serialize.

//go:generate protoc --plugin=protoc-gen-go=../../scripts/protoc/protoc-gen-go --plugin=protoc-gen-rangerrpc=../../scripts/protoc/protoc-gen-rangerrpc --proto_path=../../:../../mql:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. scan.proto

type AssetJob struct {
	DoRecord         bool
	UpstreamConfig   *upstream.UpstreamConfig
	Asset            *inventory.Asset
	Bundle           *policy.Bundle
	PolicyFilters    []string
	Props            map[string]string
	Ctx              context.Context
	Reporter         Reporter
	runtime          *providers.Runtime
	ProgressReporter progress.Progress

	// BundleCompileCache reuses the compiled bundle across the assets of one
	// scan. It is optional. When it is nil the bundle is compiled per asset.
	BundleCompileCache *bundleCompileCache
}
