// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"strconv"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/mql"
	"go.mondoo.com/mql/exec"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/mqlc"
	"go.mondoo.com/mql/providers"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// RuntimeLike is the connected runtime returned by ConnectTarget/ConnectRecording,
// aliased so callers need not import llx directly.
type RuntimeLike = llx.Runtime

// RuntimeRunner executes MQL against a connected provider runtime. It backs the
// execute-and-assert validator when a --test-target is supplied.
type RuntimeRunner struct {
	runtime  llx.Runtime
	features mql.Features
}

// Run compiles and executes the query against the connected runtime, returning
// the boolean verdict and whether the query resolved to a concrete value. The
// request's props reach the compiler for the same reason the compile gate needs
// them: a query the prompt asked to reference props.<name> does not compile
// without them.
func (r *RuntimeRunner) Run(req ValidationRequest) (bool, bool, error) {
	conf := mqlc.NewConfig(r.runtime.Schema(), r.features)
	bundle, err := mqlc.Compile(req.MQL, newPropsHandler(req.Props, conf), conf)
	if err != nil {
		return false, false, errors.Wrap(err, "failed to compile")
	}

	results, err := exec.ExecuteCode(r.runtime, bundle, nil, r.features)
	if err != nil {
		return false, false, err
	}

	raw := lastEntrypointResult(bundle, results)
	if raw == nil || raw.Data == nil {
		return false, false, nil // no result -> unresolved
	}
	if raw.Data.Error != nil {
		return false, false, raw.Data.Error
	}
	// IsTruthy returns (value, valid); valid is false for a null/unresolved
	// result, which is exactly the failure we want execute-and-assert to catch.
	value, valid := raw.Data.IsTruthy()
	return value, valid, nil
}

// lastEntrypointResult returns the result for the query's final entrypoint (the
// top-level verdict expression), mirroring how the test harness extracts a
// query's outcome.
func lastEntrypointResult(bundle *llx.CodeBundle, results map[string]*llx.RawResult) *llx.RawResult {
	if bundle == nil || bundle.CodeV2 == nil {
		return nil
	}
	entrypoints := bundle.CodeV2.Entrypoints()
	if len(entrypoints) == 0 {
		return nil
	}
	ref := entrypoints[len(entrypoints)-1]
	checksum := bundle.CodeV2.Checksums[ref]
	if r, ok := results[checksum]; ok {
		return r
	}
	// fall back to any entrypoint result
	for _, ep := range entrypoints {
		if r, ok := results[bundle.CodeV2.Checksums[ep]]; ok {
			return r
		}
	}
	return nil
}

// ConnectTarget connects a provider runtime to a target for execute-and-assert.
// target is a connection type/provider name such as "local", "aws", "gcp", or
// "azure"; the provider connects with its ambient defaults (e.g. the local AWS
// credentials, exactly like `cnspec scan aws`). Richer connection strings (ssh
// host arguments, docker image references) are not yet parsed here.
//
// It returns the connected runtime and a cleanup function that releases it.
func ConnectTarget(target string) (llx.Runtime, func(), error) {
	if target == "" {
		return nil, nil, errors.New("no test target specified")
	}
	return connectAsset(&inventory.Asset{
		Connections: []*inventory.Config{{Type: target}},
	}, "target "+strconv.Quote(target))
}

// ConnectRecording connects a runtime that replays a recording file, so
// execute-and-assert can run reproducibly with no live credentials or network.
// The recording provider is built in, so this works even when the target
// provider's binary is not installed.
func ConnectRecording(path string) (llx.Runtime, func(), error) {
	if path == "" {
		return nil, nil, errors.New("no recording path specified")
	}
	return connectAsset(&inventory.Asset{
		Connections: []*inventory.Config{{Type: "recording", Path: path}},
	}, "recording "+strconv.Quote(path))
}

// connectAsset selects a provider for the asset, connects it, and returns the
// runtime plus a cleanup that releases it. label is used in error messages.
func connectAsset(asset *inventory.Asset, label string) (llx.Runtime, func(), error) {
	runtime, err := providers.Coordinator.RuntimeFor(asset, providers.DefaultRuntime())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not select a provider for %s", label)
	}

	if err := runtime.Connect(&plugin.ConnectReq{
		Asset:    asset,
		Features: []byte(mql.DefaultFeatures),
	}); err != nil {
		// release the runtime we just acquired so a failed connect doesn't leak it
		providers.Coordinator.RemoveRuntime(runtime)
		return nil, nil, errors.Wrapf(err, "could not connect to %s", label)
	}

	cleanup := func() { providers.Coordinator.RemoveRuntime(runtime) }
	return runtime, cleanup, nil
}

// NewRuntimeRunner builds a QueryRunner over a connected runtime.
func NewRuntimeRunner(runtime llx.Runtime) *RuntimeRunner {
	return &RuntimeRunner{runtime: runtime, features: mql.DefaultFeatures}
}
