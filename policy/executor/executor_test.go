package executor

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers/mock"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/resources"
	resource_pack "go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/types"
	"go.mondoo.com/cnspec"
)

func initExecutor() *Executor {
	transport, err := mock.NewFromTomlFile("./testdata/arch.toml")
	if err != nil {
		panic(err.Error())
	}

	motor, err := motor.New(transport)
	if err != nil {
		panic(err.Error())
	}

	registry := resource_pack.Registry
	runtime := resources.NewRuntime(registry, motor)
	executor := New(registry.Schema(), runtime)

	return executor
}

type value struct {
	max   int
	err   error
	value interface{}
}

func runTest(t *testing.T, code string, expected map[string]value, callers ...func(*testing.T, *Executor)) {
	t.Run(code, func(t *testing.T) {
		received := map[string]int{}

		executor := initExecutor()
		executor.AddWatcher("default", func(res *llx.RawResult) {
			cur, ok := expected[res.CodeID]
			if !ok {
				t.Error("received an unexpected result: codeID=" + res.CodeID)
				return
			}

			assert.Equal(t, cur.err, res.Data.Error, "codeID="+res.CodeID)
			assert.Equal(t, cur.value, res.Data.Value, "codeID="+res.CodeID)
			received[res.CodeID]++
		})
		executor.AddCode(code, nil)

		ok := executor.WaitForResults(2 * time.Second)
		if !ok {
			t.Error("failed to receive results without timeout (code=" + code + ")")
		}

		for k, v := range expected {
			receivedCnt, ok := received[k]
			if !ok {
				t.Error("received no results for entrypoint " + k)
				continue
			}

			if receivedCnt != v.max {
				t.Errorf("number of received results did not match: expected=%d received=%d  for  entrypoint=%s\n", v.max, receivedCnt, k)
			}
		}

		for i := range callers {
			callers[i](t, executor)
		}
	})

	t.Run("codeBundle: "+code, func(t *testing.T) {
		received := map[string]int{}

		executor := initExecutor()
		executor.AddWatcher("default", func(res *llx.RawResult) {
			cur, ok := expected[res.CodeID]
			if !ok {
				t.Error("received an unexpected result: codeID=" + res.CodeID)
				return
			}

			assert.Equal(t, cur.err, res.Data.Error, "codeID="+res.CodeID)
			assert.Equal(t, cur.value, res.Data.Value, "codeID="+res.CodeID)
			received[res.CodeID]++
		})

		codeBundle, err := mqlc.Compile(code, resource_pack.Registry.Schema(), nil, nil)
		require.NoError(t, err)
		executor.AddCodeBundle(codeBundle, nil)

		ok := executor.WaitForResults(2 * time.Second)
		if !ok {
			t.Error("failed to receive results without timeout (code=" + code + ")")
		}

		for k, v := range expected {
			receivedCnt, ok := received[k]
			if !ok {
				t.Error("received no results for entrypoint " + k)
				continue
			}

			if receivedCnt != v.max {
				t.Errorf("number of received results did not match: expected=%d received=%d  for  entrypoint=%s\n", v.max, receivedCnt, k)
			}
		}

		for i := range callers {
			callers[i](t, executor)
		}
	})
}

func TestExecutor(t *testing.T) {
	runTest(t, "", map[string]value{
		"JSMihOSc8ss=": {
			1, nil, nil,
		},
	})

	runTest(t, "package('not').installed == false", map[string]value{
		"olBgIHiECeDWquxQNId+6HvPuwUm+GgWNyZFv3qBfbpFA5I6nKEVSX8ynKw0DUc+ijW+D1hcpBheELESIbDTdA==": {
			2, nil, false,
		},
		"a15HA8C3jENBZ+X5vgqz3/octJmFOANb1n5dVyefrHSAvY4oyU/gigll79skqGHVn82I+hduvsoTRV43qOejLA==": {
			1, nil, true,
		},
		"r427tRVa5cg=": {
			1, nil, true,
		},
	})
}

func TestUnknownResource(t *testing.T) {
	executor := initExecutor()
	var recordedErr error
	lock := sync.Mutex{}
	executor.AddWatcher("default", func(res *llx.RawResult) {
		if res.CodeID == "SuUuPeRFaKe=" {
			lock.Lock()
			defer lock.Unlock()
			recordedErr = res.Data.Error
		}
	})

	err := executor.AddCodeBundle(&llx.CodeBundle{
		Source: "fakey.mcfakerson",
		DeprecatedV5Code: &llx.CodeV1{
			Id: "SuUuPeRFaKe=",
			Checksums: map[int32]string{
				1: "fakeychecksum==",
				2: "mcfakersonchecksum==",
			},
			Code: []*llx.Chunk{
				{
					Call: llx.Chunk_FUNCTION,
					Id:   "fakey",
				},
				{
					Call: llx.Chunk_FUNCTION,
					Function: &llx.Function{
						DeprecatedV5Binding: 1,
						Type:                string(types.Bool),
					},
					Id: "mcfakerson",
				},
			},
			Entrypoints: []int32{
				2,
			},
		},
	}, nil)
	require.NoError(t, err)

	ok := executor.WaitForResults(2 * time.Second)
	require.True(t, ok)

	require.Error(t, recordedErr)
	require.Contains(t, recordedErr.Error(), "cannot find resource 'fakey'")
}

func TestMinMondooVersion(t *testing.T) {
	cnspec.Version = "5.14.0"
	executor := initExecutor()
	var recordedErr error
	lock := sync.Mutex{}
	executor.AddWatcher("default", func(res *llx.RawResult) {
		if res.CodeID == "SuUuPeRFaKe=" {
			lock.Lock()
			defer lock.Unlock()
			recordedErr = res.Data.Error
		}
	})

	err := executor.AddCodeBundle(&llx.CodeBundle{
		Source: "fakey.mcfakerson",
		DeprecatedV5Code: &llx.CodeV1{
			Id: "SuUuPeRFaKe=",
			Checksums: map[int32]string{
				1: "fakeychecksum==",
				2: "mcfakersonchecksum==",
			},
			Code: []*llx.Chunk{
				{
					Call: llx.Chunk_FUNCTION,
					Id:   "fakey",
				},
				{
					Call: llx.Chunk_FUNCTION,
					Function: &llx.Function{
						DeprecatedV5Binding: 1,
						Type:                string(types.Bool),
					},
					Id: "mcfakerson",
				},
			},
			Entrypoints: []int32{
				2,
			},
		},
		MinMondooVersion: "999.999.999",
	}, nil)

	require.NoError(t, err)

	ok := executor.WaitForResults(2 * time.Second)
	require.True(t, ok)

	require.Error(t, recordedErr)
	require.Contains(t, recordedErr.Error(), "Unable to run query, mondoo client version 999.999.999 required")
}

func TestMinMondooVersionLocal(t *testing.T) {
	cnspec.Version = "unstable"
	executor := initExecutor()
	var recordedErr error
	lock := sync.Mutex{}
	executor.AddWatcher("default", func(res *llx.RawResult) {
		if res.CodeID == "SuUuPeRFaKe=" {
			lock.Lock()
			defer lock.Unlock()
			recordedErr = res.Data.Error
		}
	})

	err := executor.AddCodeBundle(&llx.CodeBundle{
		Source: "fakey.mcfakerson",
		DeprecatedV5Code: &llx.CodeV1{
			Id: "SuUuPeRFaKe=",
			Checksums: map[int32]string{
				1: "fakeychecksum==",
				2: "mcfakersonchecksum==",
			},
			Code: []*llx.Chunk{
				{
					Call: llx.Chunk_FUNCTION,
					Id:   "fakey",
				},
				{
					Call: llx.Chunk_FUNCTION,
					Function: &llx.Function{
						DeprecatedV5Binding: 1,
						Type:                string(types.Bool),
					},
					Id: "mcfakerson",
				},
			},
			Entrypoints: []int32{
				2,
			},
		},
		MinMondooVersion: "999.999.999",
	}, nil)

	require.NoError(t, err)

	ok := executor.WaitForResults(2 * time.Second)
	require.True(t, ok)

	require.Error(t, recordedErr)
	require.Contains(t, recordedErr.Error(), "cannot find resource 'fakey")
}

func TestMinMondooVersionMissing(t *testing.T) {
	executor := initExecutor()
	var recordedErr error
	lock := sync.Mutex{}
	executor.AddWatcher("default", func(res *llx.RawResult) {
		if res.CodeID == "SuUuPeRFaKe=" {
			lock.Lock()
			defer lock.Unlock()
			recordedErr = res.Data.Error
		}
	})

	err := executor.AddCodeBundle(&llx.CodeBundle{
		Source: "fakey.mcfakerson",
		DeprecatedV5Code: &llx.CodeV1{
			Id: "SuUuPeRFaKe=",
			Checksums: map[int32]string{
				1: "fakeychecksum==",
				2: "mcfakersonchecksum==",
			},
			Code: []*llx.Chunk{
				{
					Call: llx.Chunk_FUNCTION,
					Id:   "fakey",
				},
				{
					Call: llx.Chunk_FUNCTION,
					Function: &llx.Function{
						DeprecatedV5Binding: 1,
						Type:                string(types.Bool),
					},
					Id: "mcfakerson",
				},
			},
			Entrypoints: []int32{
				2,
			},
		},
	}, nil)

	require.NoError(t, err)

	ok := executor.WaitForResults(2 * time.Second)
	require.True(t, ok)

	require.Error(t, recordedErr)
	require.Contains(t, recordedErr.Error(), "cannot find resource 'fakey")
}
