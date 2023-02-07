package scan

import (
	"context"
	"math/rand"
	"time"

	"go.mondoo.com/cnquery/cli/progress"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnspec/policy"
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
	UpstreamConfig   resources.UpstreamConfig
	Asset            *asset.Asset
	Bundle           *policy.Bundle
	PolicyFilters    []string
	Props            map[string]string
	Ctx              context.Context
	CredsResolver    vault.Resolver
	Reporter         Reporter
	connection       *motor.Motor
	ProgressReporter progress.Progress
}
