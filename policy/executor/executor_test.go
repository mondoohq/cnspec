// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package executor

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v9"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/mqlc"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/testutils"
	"go.mondoo.com/cnquery/v9/types"
	"go.mondoo.com/cnspec/v9"
)

func initExecutor() *Executor {
	runtime := testutils.LinuxMock()
	executor := New(runtime)

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

		codeBundle, err := mqlc.Compile(code, nil, mqlc.NewConfig(executor.Schema(), cnquery.DefaultFeatures))
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
		"+tMUN+5YkDw=": {
			1, nil, nil,
		},
	})

	runTest(t, "package('acl').installed == true", map[string]value{
		"IA/mh1qcKcrnANOhYpgeYqtqFWe7od9D8L1rskL+LmySCnOHnLjaQww2MZL+lhEVcE9vz8+IRM9YAxSCRJ2iwA==": {
			2, nil, true,
		},
		"NRSGjPzTnDC5EeUFEAe0LaM9MtNtgkiq/D8lhxx0TTtKb9IULE672Tfe7N9smyqjs/hdWobucKNsWnkvS6JJ9A==": {
			1, nil, true,
		},
		"4Q1qtmgoTTk=": {
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
		CodeV2: &llx.CodeV2{
			Id: "SuUuPeRFaKe=",
			Checksums: map[uint64]string{
				(1<<32 | 1): "fakeychecksum==",
				(1<<32 | 2): "mcfakersonchecksum==",
			},
			Blocks: []*llx.Block{
				{
					Chunks: []*llx.Chunk{
						{
							Call: llx.Chunk_FUNCTION,
							Id:   "fakey",
						},
						{
							Call: llx.Chunk_FUNCTION,
							Function: &llx.Function{
								Binding: (1<<32 | 1),
								Type:    string(types.Bool),
							},
							Id: "mcfakerson",
						},
					},
					Entrypoints: []uint64{
						(1<<32 | 2),
					},
				},
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
		CodeV2: &llx.CodeV2{
			Id: "SuUuPeRFaKe=",
			Checksums: map[uint64]string{
				(1<<32 | 1): "fakeychecksum==",
				(1<<32 | 2): "mcfakersonchecksum==",
			},
			Blocks: []*llx.Block{
				{
					Chunks: []*llx.Chunk{
						{
							Call: llx.Chunk_FUNCTION,
							Id:   "fakey",
						},
						{
							Call: llx.Chunk_FUNCTION,
							Function: &llx.Function{
								Binding: (1<<32 | 1),
								Type:    string(types.Bool),
							},
							Id: "mcfakerson",
						},
					},
					Entrypoints: []uint64{
						(1<<32 | 2),
					},
				},
			},
		},
		MinMondooVersion: "999.999.999",
	}, nil)

	require.NoError(t, err)

	ok := executor.WaitForResults(2 * time.Second)
	require.True(t, ok)

	require.Error(t, recordedErr)
	require.Contains(t, recordedErr.Error(), "Unable to run query, cnspec version 999.999.999 required")
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
		CodeV2: &llx.CodeV2{
			Id: "SuUuPeRFaKe=",
			Checksums: map[uint64]string{
				(1<<32 | 1): "fakeychecksum==",
				(1<<32 | 2): "mcfakersonchecksum==",
			},
			Blocks: []*llx.Block{
				{
					Chunks: []*llx.Chunk{
						{
							Call: llx.Chunk_FUNCTION,
							Id:   "fakey",
						},
						{
							Call: llx.Chunk_FUNCTION,
							Function: &llx.Function{
								Binding: (1<<32 | 1),
								Type:    string(types.Bool),
							},
							Id: "mcfakerson",
						},
					},
					Entrypoints: []uint64{
						(1<<32 | 2),
					},
				},
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
		CodeV2: &llx.CodeV2{
			Id: "SuUuPeRFaKe=",
			Checksums: map[uint64]string{
				(1<<32 | 1): "fakeychecksum==",
				(1<<32 | 2): "mcfakersonchecksum==",
			},
			Blocks: []*llx.Block{
				{
					Chunks: []*llx.Chunk{
						{
							Call: llx.Chunk_FUNCTION,
							Id:   "fakey",
						},
						{
							Call: llx.Chunk_FUNCTION,
							Function: &llx.Function{
								Binding: (1<<32 | 1),
								Type:    string(types.Bool),
							},
							Id: "mcfakerson",
						},
					},
					Entrypoints: []uint64{
						(1<<32 | 2),
					},
				},
			},
		},
	}, nil)

	require.NoError(t, err)

	ok := executor.WaitForResults(2 * time.Second)
	require.True(t, ok)

	require.Error(t, recordedErr)
	require.Contains(t, recordedErr.Error(), "cannot find resource 'fakey")
}
