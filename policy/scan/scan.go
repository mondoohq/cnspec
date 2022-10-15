package scan

import (
	"context"
	"math/rand"
	"time"

	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnspec/policy"
)

// Why do we want the scan stuff as a proto? Because we have a disk queue
// and it makes it easier and faster to serialize.

//go:generate protoc -I ../../cnquery --proto_path=../../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. cnspec_policy_scan.proto

// 50MB default size
const ResolvedPolicyCacheSize = 52428800

func init() {
	rand.Seed(time.Now().UnixNano())
}

type AssetJob struct {
	DoRecord      bool
	Incognito     bool
	Asset         *asset.Asset
	Bundle        *policy.Bundle
	PolicyFilters []string
	Ctx           context.Context
	GetCredential func(cred *vault.Credential) (*vault.Credential, error)
	Reporter      Reporter
	connection    *motor.Motor
}
