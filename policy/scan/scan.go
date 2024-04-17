// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"math/rand"
	"time"

	"go.mondoo.com/cnquery/v11/cli/progress"
	"go.mondoo.com/cnquery/v11/providers"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream"
	"go.mondoo.com/cnspec/v11/policy"
)

// Why do we want the scan stuff as a proto? Because we have a disk queue
// and it makes it easier and faster to serialize.

//go:generate protoc --proto_path=../../:../../cnquery:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. scan.proto

// 50MB default size
const ResolvedPolicyCacheSize = 52428800

func init() {
	rand.Seed(time.Now().UnixNano())
}

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
}
