package internal

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/types"
	"go.mondoo.com/cnspec/policy"
)

func TestDatapointNode(t *testing.T) {
	newNodeData := func() *DatapointNodeData {
		return &DatapointNodeData{}
	}
	t.Run("initialize/recalculate", func(t *testing.T) {
		t.Run("does not recalculate if data is not provided", func(t *testing.T) {
			nodeData := newNodeData()

			nodeData.initialize()
			data := nodeData.recalculate()

			assert.Nil(t, data)
		})

		t.Run("recalculates if data is provided", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.res = &llx.RawResult{
				CodeID: "checksum",
				Data:   llx.BoolTrue,
			}

			nodeData.initialize()
			data := nodeData.recalculate()

			require.NotNil(t, data)
			require.NotNil(t, data.res)
			assert.Equal(t, "checksum", data.res.CodeID)
			assert.Equal(t, llx.BoolTrue, data.res.Data)
		})

		t.Run("casts if required type is provided", func(t *testing.T) {
			nodeData := newNodeData()
			typ := string(types.Bool)
			nodeData.expectedType = &typ
			nodeData.res = &llx.RawResult{
				CodeID: "checksum",
				Data:   llx.StringData("hello"),
			}

			nodeData.initialize()
			data := nodeData.recalculate()

			require.NotNil(t, data)
			require.NotNil(t, data.res)
			assert.Equal(t, "checksum", data.res.CodeID)
			assert.Equal(t, llx.BoolTrue, data.res.Data)
		})
	})

	t.Run("consume/recalculate", func(t *testing.T) {
		t.Run("ignores nils", func(t *testing.T) {
			nodeData := newNodeData()

			nodeData.initialize()
			nodeData.recalculate()

			nodeData.consume(NodeID("__executor__"), &envelope{})
			data := nodeData.recalculate()
			assert.Nil(t, data)
		})

		t.Run("recalculate when data arrives", func(t *testing.T) {
			nodeData := newNodeData()

			nodeData.initialize()
			nodeData.recalculate()

			nodeData.consume(NodeID("__executor__"), &envelope{
				res: &llx.RawResult{
					CodeID: "checksum",
					Data:   llx.BoolTrue,
				},
			})
			data := nodeData.recalculate()

			require.NotNil(t, data)
			require.NotNil(t, data.res)
			assert.Equal(t, "checksum", data.res.CodeID)
			assert.Equal(t, llx.BoolTrue, data.res.Data)
		})

		t.Run("doesn't recalculate multiple times", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.res = &llx.RawResult{
				CodeID: "checksum",
				Data:   llx.BoolTrue,
			}

			nodeData.initialize()
			data := nodeData.recalculate()
			require.NotNil(t, data)
			assert.NotNil(t, data.res)

			nodeData.consume(NodeID("__executor__"), &envelope{
				res: &llx.RawResult{
					CodeID: "checksum",
					Data:   llx.BoolFalse,
				},
			})
			data = nodeData.recalculate()
			assert.Nil(t, data)
		})

		t.Run("casts if required type is provided", func(t *testing.T) {
			nodeData := newNodeData()
			typ := string(types.Bool)
			nodeData.expectedType = &typ

			nodeData.initialize()
			nodeData.recalculate()

			nodeData.consume(NodeID("__executor__"), &envelope{
				res: &llx.RawResult{
					CodeID: "checksum",
					Data:   llx.StringData("hello"),
				},
			})
			data := nodeData.recalculate()

			require.NotNil(t, data)
			require.NotNil(t, data.res)
			assert.Equal(t, "checksum", data.res.CodeID)
			assert.Equal(t, llx.BoolTrue, data.res.Data)
		})

		t.Run("skips cast if required type are same", func(t *testing.T) {
			nodeData := newNodeData()
			typ := string(types.String)
			nodeData.expectedType = &typ

			nodeData.initialize()
			nodeData.recalculate()

			resData := llx.StringData("hello")
			nodeData.consume(NodeID("__executor__"), &envelope{
				res: &llx.RawResult{
					CodeID: "checksum",
					Data:   resData,
				},
			})
			data := nodeData.recalculate()

			require.NotNil(t, data)
			require.NotNil(t, data.res)
			assert.Equal(t, "checksum", data.res.CodeID)
			assert.Equal(t, resData, data.res.Data)
		})

		t.Run("skips cast if datapoint is error", func(t *testing.T) {
			nodeData := newNodeData()
			typ := string(types.String)
			nodeData.expectedType = &typ

			nodeData.initialize()
			nodeData.recalculate()

			nodeData.consume(NodeID("__executor__"), &envelope{
				res: &llx.RawResult{
					CodeID: "checksum",
					Data: &llx.RawData{
						Error: errors.New("error happened"),
					},
				},
			})
			data := nodeData.recalculate()

			require.NotNil(t, data)
			require.NotNil(t, data.res)
			assert.Equal(t, "checksum", data.res.CodeID)
			require.NotNil(t, data.res.Data.Error)
			assert.Equal(t, "error happened", data.res.Data.Error.Error())
			assert.Nil(t, data.res.Data.Value)
		})

		t.Run("skips cast if expected type is unset", func(t *testing.T) {
			nodeData := newNodeData()
			typ := string(types.Unset)
			nodeData.expectedType = &typ

			nodeData.initialize()
			nodeData.recalculate()

			resData := llx.StringData("hello")
			nodeData.consume(NodeID("__executor__"), &envelope{
				res: &llx.RawResult{
					CodeID: "checksum",
					Data:   resData,
				},
			})
			data := nodeData.recalculate()

			require.NotNil(t, data)
			require.NotNil(t, data.res)
			assert.Equal(t, "checksum", data.res.CodeID)
			assert.Equal(t, resData, data.res.Data)
		})
	})
}

func TestExecutionQueryNode(t *testing.T) {
	newNodeData := func() (*ExecutionQueryNodeData, chan runQueueItem) {
		q := make(chan runQueueItem, 1)
		data := &ExecutionQueryNodeData{
			queryID:            "testqueryid",
			requiredProperties: map[string]*executionQueryProperty{},
			runState:           notReadyQueryNotReady,
			runQueue:           q,
			codeBundle: &llx.CodeBundle{
				CodeV2: &llx.CodeV2{
					Id: "testqueryid",
				},
			},
		}
		return data, q
	}
	t.Run("initialize/recalculate", func(t *testing.T) {
		t.Run("does not recalculate if dependencies not satisfied", func(t *testing.T) {
			nodeData, q := newNodeData()
			nodeData.requiredProperties = map[string]*executionQueryProperty{
				"prop1": {
					name:     "prop1",
					checksum: "checksum1",
					resolved: false,
				},
			}
			nodeData.initialize()
			data := nodeData.recalculate()
			assert.Nil(t, data)
			select {
			case <-q:
				assert.Fail(t, "not ready for exectuion")
			default:
			}
		})
		t.Run("recalculates if dependencies are satisfied", func(t *testing.T) {
			nodeData, q := newNodeData()
			nodeData.requiredProperties = map[string]*executionQueryProperty{
				"prop1": {
					name:     "prop1",
					checksum: "checksum1",
					resolved: true,
					value:    llx.BoolFalse.Result(),
				},
				"prop2": {
					name:     "prop2",
					checksum: "checksum1",
					resolved: true,
					value:    llx.BoolFalse.Result(),
				},
			}
			nodeData.initialize()
			data := nodeData.recalculate()
			assert.NotNil(t, data)
			assert.Nil(t, data.res)
			assert.Nil(t, data.score)
			select {
			case item := <-q:
				require.NotNil(t, item.codeBundle)
				assert.Equal(t, "testqueryid", item.codeBundle.CodeV2.Id)
				assert.Contains(t, item.props, "prop1")
			default:
				assert.Fail(t, "expected something to be executed")
			}
		})
	})

	t.Run("consume/recalculate", func(t *testing.T) {
		t.Run("does not recalculate if dependencies not satisfied", func(t *testing.T) {
			nodeData, q := newNodeData()
			nodeData.requiredProperties = map[string]*executionQueryProperty{
				"prop1": {
					name:     "prop1",
					checksum: "checksum1",
				},
				"prop2": {
					name:     "prop2",
					checksum: "checksum2",
				},
			}
			nodeData.initialize()
			data := nodeData.recalculate()
			assert.Nil(t, data)
			nodeData.consume(NodeID("checksum1"), &envelope{
				res: &llx.RawResult{
					CodeID: "checksum1",
					Data:   llx.BoolTrue,
				},
			})

			select {
			case <-q:
				assert.Fail(t, "not ready for exectuion")
			default:
			}
		})
		t.Run("only recalculates once", func(t *testing.T) {
			nodeData, q := newNodeData()
			nodeData.requiredProperties = map[string]*executionQueryProperty{
				"prop1": {
					name:     "prop1",
					checksum: "checksum1",
				},
				"prop2": {
					name:     "prop2",
					checksum: "checksum1",
				},
			}
			nodeData.initialize()
			data := nodeData.recalculate()
			assert.Nil(t, data)
			nodeData.consume(NodeID("checksum1"), &envelope{
				res: &llx.RawResult{
					CodeID: "checksum1",
					Data:   llx.BoolTrue,
				},
			})
			data = nodeData.recalculate()
			assert.NotNil(t, data)
			select {
			case _ = <-q:
			default:
				assert.Fail(t, "expected something to be executed")
			}

			nodeData.consume(NodeID("checksum1"), &envelope{
				res: &llx.RawResult{
					CodeID: "checksum1",
					Data:   llx.BoolTrue,
				},
			})
			data = nodeData.recalculate()
			select {
			case _ = <-q:
				assert.Fail(t, "query should not re-execute")
			default:
			}
		})
		t.Run("recalculates after all dependencies are satisfied", func(t *testing.T) {})
	})
}

func TestReportingQueryNode(t *testing.T) {
	newNodeData := func() *ReportingQueryNodeData {
		data := &ReportingQueryNodeData{
			queryID: "testqueryid",
		}
		return data
	}
	t.Run("initialize/recalculate", func(t *testing.T) {
		t.Run("recalculates if there are no dependencies", func(t *testing.T) {
			nodeData := newNodeData()

			nodeData.initialize()
			data := nodeData.recalculate()

			require.NotNil(t, data)
			assert.Nil(t, data.res)
			require.NotNil(t, data.score)
			assert.Equal(t, "testqueryid", data.score.QrId)
			assert.Equal(t, policy.ScoreType_Skip, data.score.Type)
			assert.Equal(t, 100, int(data.score.ScoreCompletion))

			data = nodeData.recalculate()
			assert.Nil(t, data)
		})
		t.Run("recalculates if all dependencies are satisfied", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.results = map[string]*DataResult{
				"checksum1": {
					checksum: "checksum1",
					resolved: true,
					value:    llx.BoolTrue.Result().RawResultV2(),
				},
			}

			nodeData.initialize()
			data := nodeData.recalculate()

			require.NotNil(t, data)
			assert.Nil(t, data.res)
			require.NotNil(t, data.score)
			assert.Equal(t, "testqueryid", data.score.QrId)
			assert.Equal(t, policy.ScoreType_Result, data.score.Type)
			assert.Equal(t, 100, int(data.score.Value))
			assert.Equal(t, 100, int(data.score.ScoreCompletion))

			data = nodeData.recalculate()
			assert.Nil(t, data)
		})
		t.Run("does not recalculate if any dependencies are missing", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.results = map[string]*DataResult{
				"checksum1": {
					checksum: "checksum1",
				},
			}

			nodeData.initialize()
			data := nodeData.recalculate()

			require.Nil(t, data)
		})
	})

	t.Run("consume/recalculate", func(t *testing.T) {
		t.Run("does not recalculates if any dependencies have not been resolved", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.results = map[string]*DataResult{
				"checksum1": {
					checksum: "checksum1",
				},
				"checksum2": {
					checksum: "checksum2",
				},
			}

			nodeData.initialize()
			nodeData.recalculate()

			nodeData.consume(NodeID("checksum1"), &envelope{
				res: llx.BoolTrue.Result().RawResultV2(),
			})

			data := nodeData.recalculate()
			assert.Nil(t, data)
		})
		t.Run("recalculates if all dependencies have been resolved", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.results = map[string]*DataResult{
				"checksum1": {
					checksum: "checksum1",
				},
				"checksum2": {
					checksum: "checksum2",
				},
			}

			nodeData.initialize()
			nodeData.recalculate()

			nodeData.consume(NodeID("checksum1"), &envelope{
				res: llx.BoolTrue.Result().RawResultV2(),
			})
			nodeData.consume(NodeID("checksum2"), &envelope{
				res: llx.BoolTrue.Result().RawResultV2(),
			})

			data := nodeData.recalculate()
			require.NotNil(t, data)
			require.NotNil(t, data.score)
			assert.Equal(t, "testqueryid", data.score.QrId)
			assert.Equal(t, policy.ScoreType_Result, data.score.Type)
			assert.Equal(t, 100, int(data.score.Value))
			assert.Equal(t, 100, int(data.score.ScoreCompletion))
		})
		t.Run("does not recalculate after completion", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.results = map[string]*DataResult{
				"checksum1": {
					checksum: "checksum1",
				},
				"checksum2": {
					checksum: "checksum2",
				},
			}

			nodeData.initialize()
			nodeData.recalculate()

			nodeData.consume(NodeID("checksum1"), &envelope{
				res: llx.BoolTrue.Result().RawResultV2(),
			})
			nodeData.consume(NodeID("checksum2"), &envelope{
				res: llx.BoolTrue.Result().RawResultV2(),
			})

			data := nodeData.recalculate()
			require.NotNil(t, data)
			require.NotNil(t, data.score)

			nodeData.consume(NodeID("checksum2"), &envelope{
				res: llx.BoolFalse.Result().RawResultV2(),
			})
			data = nodeData.recalculate()
			require.Nil(t, data)
		})
	})

	t.Run("test scoring", func(t *testing.T) {
		t.Run("is error if any dependencies are error", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.results = map[string]*DataResult{
				"checksum1": {
					checksum: "checksum1",
					resolved: true,
					value:    llx.BoolTrue.Result().RawResultV2(),
				},
				"checksum2": {
					checksum: "checksum2",
				},
			}

			nodeData.initialize()
			data := nodeData.recalculate()
			assert.Nil(t, data)

			nodeData.consume(NodeID("checksum2"), &envelope{
				res: &llx.RawResult{
					CodeID: "checksum2",
					Data: &llx.RawData{
						Error: errors.New("error"),
					},
				},
			})

			data = nodeData.recalculate()
			require.NotNil(t, data)
			require.NotNil(t, data.score)
			assert.Equal(t, "testqueryid", data.score.QrId)
			assert.Equal(t, policy.ScoreType_Error, data.score.Type)
			assert.Equal(t, 100, int(data.score.ScoreCompletion))
		})
		t.Run("skipped if all nil", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.results = map[string]*DataResult{
				"checksum1": {
					checksum: "checksum1",
					resolved: true,
					value:    llx.NilData.Result().RawResultV2(),
				},
				"checksum2": {
					checksum: "checksum2",
				},
			}

			nodeData.initialize()
			data := nodeData.recalculate()
			assert.Nil(t, data)

			nodeData.consume(NodeID("checksum2"), &envelope{
				res: &llx.RawResult{
					CodeID: "checksum2",
					Data:   llx.NilData,
				},
			})

			data = nodeData.recalculate()
			require.NotNil(t, data)
			require.NotNil(t, data.score)
			assert.Equal(t, "testqueryid", data.score.QrId)
			assert.Equal(t, policy.ScoreType_Skip, data.score.Type)
			assert.Equal(t, 100, int(data.score.ScoreCompletion))
		})
		t.Run("result if all dependencies satisfied", func(t *testing.T) {
			t.Run("100 if all true", func(t *testing.T) {
				nodeData := newNodeData()
				nodeData.results = map[string]*DataResult{
					"checksum1": {
						checksum: "checksum1",
						resolved: true,
						value:    llx.BoolTrue.Result().RawResultV2(),
					},
					"checksum2": {
						checksum: "checksum2",
					},
				}

				nodeData.initialize()
				data := nodeData.recalculate()
				assert.Nil(t, data)

				nodeData.consume(NodeID("checksum2"), &envelope{
					res: &llx.RawResult{
						CodeID: "checksum2",
						Data:   llx.BoolTrue,
					},
				})

				data = nodeData.recalculate()
				require.NotNil(t, data)
				require.NotNil(t, data.score)
				assert.Equal(t, "testqueryid", data.score.QrId)
				assert.Equal(t, policy.ScoreType_Result, data.score.Type)
				assert.Equal(t, 100, int(data.score.Value))
				assert.Equal(t, 100, int(data.score.ScoreCompletion))
			})
			t.Run("0 if any false", func(t *testing.T) {
				nodeData := newNodeData()
				nodeData.results = map[string]*DataResult{
					"checksum1": {
						checksum: "checksum1",
						resolved: true,
						value:    llx.BoolFalse.Result().RawResultV2(),
					},
					"checksum2": {
						checksum: "checksum2",
					},
				}

				nodeData.initialize()
				data := nodeData.recalculate()
				assert.Nil(t, data)

				nodeData.consume(NodeID("checksum2"), &envelope{
					res: &llx.RawResult{
						CodeID: "checksum2",
						Data:   llx.BoolTrue,
					},
				})

				data = nodeData.recalculate()
				require.NotNil(t, data)
				require.NotNil(t, data.score)
				assert.Equal(t, "testqueryid", data.score.QrId)
				assert.Equal(t, policy.ScoreType_Result, data.score.Type)
				assert.Equal(t, 0, int(data.score.Value))
				assert.Equal(t, 100, int(data.score.ScoreCompletion))
			})
		})
	})
}

func TestReportingQueryNode_BoolAssertion(t *testing.T) {
	newNodeData := func() *ReportingQueryNodeData {
		data := &ReportingQueryNodeData{
			queryID: "testqueryid",
		}
		return data
	}
	t.Run("initialize/recalculate", func(t *testing.T) {
		t.Run("recalculates if there are no dependencies", func(t *testing.T) {
			nodeData := newNodeData()

			nodeData.initialize()
			data := nodeData.recalculate()

			require.NotNil(t, data)
			assert.Nil(t, data.res)
			require.NotNil(t, data.score)
			assert.Equal(t, "testqueryid", data.score.QrId)
			assert.Equal(t, policy.ScoreType_Skip, data.score.Type)
			assert.Equal(t, 100, int(data.score.ScoreCompletion))

			data = nodeData.recalculate()
			assert.Nil(t, data)
		})
		t.Run("recalculates if all dependencies are satisfied", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.results = map[string]*DataResult{
				"checksum1": {
					checksum: "checksum1",
					resolved: true,
					value:    llx.BoolTrue.Result().RawResultV2(),
				},
			}

			nodeData.initialize()
			data := nodeData.recalculate()

			require.NotNil(t, data)
			assert.Nil(t, data.res)
			require.NotNil(t, data.score)
			assert.Equal(t, "testqueryid", data.score.QrId)
			assert.Equal(t, policy.ScoreType_Result, data.score.Type)
			assert.Equal(t, 100, int(data.score.Value))
			assert.Equal(t, 100, int(data.score.ScoreCompletion))

			data = nodeData.recalculate()
			assert.Nil(t, data)
		})
		t.Run("does not recalculate if any dependencies are missing", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.results = map[string]*DataResult{
				"checksum1": {
					checksum: "checksum1",
				},
			}

			nodeData.initialize()
			data := nodeData.recalculate()

			require.Nil(t, data)
		})
	})

	t.Run("consume/recalculate", func(t *testing.T) {
		t.Run("does not recalculates if any dependencies have not been resolved", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.results = map[string]*DataResult{
				"checksum1": {
					checksum: "checksum1",
				},
				"checksum2": {
					checksum: "checksum2",
				},
			}

			nodeData.initialize()
			nodeData.recalculate()

			nodeData.consume(NodeID("checksum1"), &envelope{
				res: llx.BoolTrue.Result().RawResultV2(),
			})

			data := nodeData.recalculate()
			assert.Nil(t, data)
		})
		t.Run("recalculates if all dependencies have been resolved", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.results = map[string]*DataResult{
				"checksum1": {
					checksum: "checksum1",
				},
				"checksum2": {
					checksum: "checksum2",
				},
			}

			nodeData.initialize()
			nodeData.recalculate()

			nodeData.consume(NodeID("checksum1"), &envelope{
				res: llx.BoolTrue.Result().RawResultV2(),
			})
			nodeData.consume(NodeID("checksum2"), &envelope{
				res: llx.BoolTrue.Result().RawResultV2(),
			})

			data := nodeData.recalculate()
			require.NotNil(t, data)
			require.NotNil(t, data.score)
			assert.Equal(t, "testqueryid", data.score.QrId)
			assert.Equal(t, policy.ScoreType_Result, data.score.Type)
			assert.Equal(t, 100, int(data.score.Value))
			assert.Equal(t, 100, int(data.score.ScoreCompletion))
		})
		t.Run("does not recalculate after completion", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.results = map[string]*DataResult{
				"checksum1": {
					checksum: "checksum1",
				},
				"checksum2": {
					checksum: "checksum2",
				},
			}

			nodeData.initialize()
			nodeData.recalculate()

			nodeData.consume(NodeID("checksum1"), &envelope{
				res: llx.BoolTrue.Result().RawResultV2(),
			})
			nodeData.consume(NodeID("checksum2"), &envelope{
				res: llx.BoolTrue.Result().RawResultV2(),
			})

			data := nodeData.recalculate()
			require.NotNil(t, data)
			require.NotNil(t, data.score)

			nodeData.consume(NodeID("checksum2"), &envelope{
				res: llx.BoolFalse.Result().RawResultV2(),
			})
			data = nodeData.recalculate()
			require.Nil(t, data)
		})
	})

	t.Run("test scoring", func(t *testing.T) {
		t.Run("is error if any dependencies are error", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.results = map[string]*DataResult{
				"checksum1": {
					checksum: "checksum1",
					resolved: true,
					value:    llx.BoolTrue.Result().RawResultV2(),
				},
				"checksum2": {
					checksum: "checksum2",
				},
			}

			nodeData.initialize()
			data := nodeData.recalculate()
			assert.Nil(t, data)

			nodeData.consume(NodeID("checksum2"), &envelope{
				res: &llx.RawResult{
					CodeID: "checksum2",
					Data: &llx.RawData{
						Error: errors.New("error"),
					},
				},
			})

			data = nodeData.recalculate()
			require.NotNil(t, data)
			require.NotNil(t, data.score)
			assert.Equal(t, "testqueryid", data.score.QrId)
			assert.Equal(t, policy.ScoreType_Error, data.score.Type)
			assert.Equal(t, 100, int(data.score.ScoreCompletion))
		})
		t.Run("skipped if all nil", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.results = map[string]*DataResult{
				"checksum1": {
					checksum: "checksum1",
					resolved: true,
					value:    llx.NilData.Result().RawResultV2(),
				},
				"checksum2": {
					checksum: "checksum2",
				},
			}

			nodeData.initialize()
			data := nodeData.recalculate()
			assert.Nil(t, data)

			nodeData.consume(NodeID("checksum2"), &envelope{
				res: &llx.RawResult{
					CodeID: "checksum2",
					Data:   llx.NilData,
				},
			})

			data = nodeData.recalculate()
			require.NotNil(t, data)
			require.NotNil(t, data.score)
			assert.Equal(t, "testqueryid", data.score.QrId)
			assert.Equal(t, policy.ScoreType_Skip, data.score.Type)
			assert.Equal(t, 100, int(data.score.ScoreCompletion))
		})

		t.Run("skipped if all non bool", func(t *testing.T) {
			nodeData := newNodeData()
			nodeData.results = map[string]*DataResult{
				"checksum1": {
					checksum: "checksum1",
					resolved: true,
					value: &llx.RawResult{
						CodeID: "checksum1",
						Data:   llx.StringData(""),
					},
				},
				"checksum2": {
					checksum: "checksum2",
					resolved: true,
					value: &llx.RawResult{
						CodeID: "checksum2",
						Data:   llx.IntData(0),
					},
				},
			}

			nodeData.initialize()

			data := nodeData.recalculate()
			require.NotNil(t, data)
			require.NotNil(t, data.score)
			assert.Equal(t, "testqueryid", data.score.QrId)
			assert.Equal(t, policy.ScoreType_Skip, data.score.Type)
			assert.Equal(t, 100, int(data.score.ScoreCompletion))
		})

		t.Run("result if all dependencies satisfied", func(t *testing.T) {
			t.Run("100 if all true", func(t *testing.T) {
				nodeData := newNodeData()
				nodeData.results = map[string]*DataResult{
					"checksum1": {
						checksum: "checksum1",
						resolved: true,
						value:    llx.BoolTrue.Result().RawResultV2(),
					},
					"checksum2": {
						checksum: "checksum2",
					},
					"checksum3": {
						checksum: "checksum3",
						resolved: true,
						value: &llx.RawResult{
							CodeID: "checksum3",
							Data:   llx.StringData(""),
						},
					},
				}

				nodeData.initialize()
				data := nodeData.recalculate()
				assert.Nil(t, data)

				nodeData.consume(NodeID("checksum2"), &envelope{
					res: &llx.RawResult{
						CodeID: "checksum2",
						Data:   llx.BoolTrue,
					},
				})

				data = nodeData.recalculate()
				require.NotNil(t, data)
				require.NotNil(t, data.score)
				assert.Equal(t, "testqueryid", data.score.QrId)
				assert.Equal(t, policy.ScoreType_Result, data.score.Type)
				assert.Equal(t, 100, int(data.score.Value))
				assert.Equal(t, 100, int(data.score.ScoreCompletion))
			})
			t.Run("0 if any false", func(t *testing.T) {
				nodeData := newNodeData()
				nodeData.results = map[string]*DataResult{
					"checksum1": {
						checksum: "checksum1",
						resolved: true,
						value:    llx.BoolFalse.Result().RawResultV2(),
					},
					"checksum2": {
						checksum: "checksum2",
					},
				}

				nodeData.initialize()
				data := nodeData.recalculate()
				assert.Nil(t, data)

				nodeData.consume(NodeID("checksum2"), &envelope{
					res: &llx.RawResult{
						CodeID: "checksum2",
						Data:   llx.BoolTrue,
					},
				})

				data = nodeData.recalculate()
				require.NotNil(t, data)
				require.NotNil(t, data.score)
				assert.Equal(t, "testqueryid", data.score.QrId)
				assert.Equal(t, policy.ScoreType_Result, data.score.Type)
				assert.Equal(t, 0, int(data.score.Value))
				assert.Equal(t, 100, int(data.score.ScoreCompletion))
			})
		})
	})
}

func TestReportingJobNode(t *testing.T) {
	newNodeData := func() *ReportingJobNodeData {
		data := &ReportingJobNodeData{
			queryID: "testqueryid",
		}
		return data
	}

	t.Run("initialize/recalculate", func(t *testing.T) {
		t.Run("recalculates if there are no dependencies", func(t *testing.T) {
			nodeData := newNodeData()

			nodeData.initialize()
			data := nodeData.recalculate()

			require.NotNil(t, data)
			assert.Nil(t, data.res)
			require.NotNil(t, data.score)
			assert.Equal(t, "testqueryid", data.score.QrId)
			assert.Equal(t, policy.ScoreType_Unscored, data.score.Type)
			assert.Equal(t, 100, int(data.score.ScoreCompletion))
			assert.Equal(t, 100, int(data.score.DataCompletion))
			assert.Equal(t, 0, int(data.score.DataTotal))
		})

		t.Run("recalculates if datapoints are provided", func(t *testing.T) {
			nodeData := newNodeData()

			nodeData.childScores = map[NodeID]*reportingJobResult{
				"rjID1": {
					score: &policy.Score{
						QrId:            "qrid1",
						Type:            policy.ScoreType_Result,
						Value:           0,
						ScoreCompletion: 0,
					},
				},
			}
			nodeData.datapoints = map[NodeID]*reportingJobDatapoint{
				"checksum1": {},
				"checksum2": {
					res: llx.BoolData(true).Result().RawResultV2(),
				},
			}

			nodeData.initialize()
			data := nodeData.recalculate()

			require.NotNil(t, data)
			assert.Nil(t, data.res)
			require.NotNil(t, data.score)
			assert.Equal(t, "testqueryid", data.score.QrId)
			assert.Equal(t, policy.ScoreType_Result, data.score.Type)
			assert.Equal(t, 0, int(data.score.Value))
			assert.Equal(t, 0, int(data.score.ScoreCompletion))
			assert.Equal(t, 50, int(data.score.DataCompletion))
			assert.Equal(t, 2, int(data.score.DataTotal))
		})

		t.Run("does not recalculate if any scores missing", func(t *testing.T) {
			nodeData := newNodeData()

			nodeData.childScores = map[NodeID]*reportingJobResult{
				"rjID1": {},
			}
			nodeData.datapoints = map[NodeID]*reportingJobDatapoint{
				"checksum1": {
					res: llx.BoolData(true).Result().RawResultV2(),
				},
			}

			nodeData.initialize()
			data := nodeData.recalculate()

			require.Nil(t, data)
		})

		t.Run("consume/recalculate", func(t *testing.T) {
			t.Run("does not recalculate if no new data provided", func(t *testing.T) {
				t.Skip("unimplemented. requires diffing the existing scores and results")
			})
			t.Run("recalculates when new data arrives", func(t *testing.T) {
				t.Run("when isQuery", func(t *testing.T) {
					t.Run("when score", func(t *testing.T) {
						nodeData := newNodeData()
						nodeData.isQuery = true
						nodeData.childScores = map[NodeID]*reportingJobResult{
							nodeData.queryID: {},
						}
						nodeData.datapoints = map[NodeID]*reportingJobDatapoint{
							"checksum1": {},
						}

						nodeData.initialize()
						data := nodeData.recalculate()

						require.NotNil(t, data)
						nodeData.consume(NodeID(nodeData.queryID), &envelope{
							score: &policy.Score{
								QrId:            nodeData.queryID,
								Type:            policy.ScoreType_Result,
								Value:           100,
								ScoreCompletion: 50,
							},
						})
						data = nodeData.recalculate()

						assert.Equal(t, "testqueryid", data.score.QrId)
						assert.Equal(t, policy.ScoreType_Result, data.score.Type)
						assert.Equal(t, 100, int(data.score.Value))
						assert.Equal(t, 50, int(data.score.ScoreCompletion))
						assert.Equal(t, 0, int(data.score.DataCompletion))
						assert.Equal(t, 1, int(data.score.DataTotal))
					})
					t.Run("when result", func(t *testing.T) {
						nodeData := newNodeData()
						nodeData.isQuery = true
						nodeData.childScores = map[NodeID]*reportingJobResult{
							nodeData.queryID: {
								score: &policy.Score{
									QrId: nodeData.queryID,
									Type: policy.ScoreType_Result,
								},
							},
						}
						nodeData.datapoints = map[NodeID]*reportingJobDatapoint{
							"checksum1": {},
						}

						nodeData.initialize()
						data := nodeData.recalculate()

						require.NotNil(t, data)
						nodeData.consume(NodeID("checksum1"), &envelope{
							res: llx.BoolTrue.Result().RawResultV2(),
						})
						data = nodeData.recalculate()

						assert.Equal(t, "testqueryid", data.score.QrId)
						assert.Equal(t, policy.ScoreType_Result, data.score.Type)
						assert.Equal(t, 0, int(data.score.Value))
						assert.Equal(t, 0, int(data.score.ScoreCompletion))
						assert.Equal(t, 100, int(data.score.DataCompletion))
						assert.Equal(t, 1, int(data.score.DataTotal))
					})
				})

				t.Run("when is control", func(t *testing.T) {
					t.Run("error converted to fail", func(t *testing.T) {
						nodeData := newNodeData()
						nodeData.rjType = policy.ReportingJob_CONTROL
						nodeData.childScores = map[NodeID]*reportingJobResult{
							nodeData.queryID: {},
						}

						nodeData.initialize()
						nodeData.recalculate()

						nodeData.consume(NodeID(nodeData.queryID), &envelope{
							score: &policy.Score{
								QrId:            nodeData.queryID,
								Type:            policy.ScoreType_Error,
								Value:           0,
								ScoreCompletion: 100,
							},
						})
						data := nodeData.recalculate()

						assert.Equal(t, "testqueryid", data.score.QrId)
						assert.Equal(t, policy.ScoreType_Result, data.score.Type)
						assert.Equal(t, 0, int(data.score.Value))
						assert.Equal(t, 100, int(data.score.ScoreCompletion))
						assert.Equal(t, 100, int(data.score.DataCompletion))
						assert.Equal(t, 0, int(data.score.DataTotal))
					})
					t.Run("unscored converted to pass", func(t *testing.T) {
						nodeData := newNodeData()
						nodeData.rjType = policy.ReportingJob_CONTROL
						nodeData.childScores = map[NodeID]*reportingJobResult{
							nodeData.queryID: {},
						}

						nodeData.initialize()
						nodeData.recalculate()

						nodeData.consume(NodeID(nodeData.queryID), &envelope{
							score: &policy.Score{
								QrId:            nodeData.queryID,
								Type:            policy.ScoreType_Unscored,
								Value:           0,
								ScoreCompletion: 100,
							},
						})
						data := nodeData.recalculate()

						assert.Equal(t, "testqueryid", data.score.QrId)
						assert.Equal(t, policy.ScoreType_Result, data.score.Type)
						assert.Equal(t, 100, int(data.score.Value))
						assert.Equal(t, 100, int(data.score.ScoreCompletion))
						assert.Equal(t, 100, int(data.score.DataCompletion))
						assert.Equal(t, 0, int(data.score.DataTotal))
					})
					t.Run("skip converted to pass", func(t *testing.T) {
						nodeData := newNodeData()
						nodeData.rjType = policy.ReportingJob_CONTROL
						nodeData.childScores = map[NodeID]*reportingJobResult{
							nodeData.queryID: {},
						}

						nodeData.initialize()
						nodeData.recalculate()

						nodeData.consume(NodeID(nodeData.queryID), &envelope{
							score: &policy.Score{
								QrId:            nodeData.queryID,
								Type:            policy.ScoreType_Skip,
								Value:           0,
								ScoreCompletion: 100,
							},
						})
						data := nodeData.recalculate()

						assert.Equal(t, "testqueryid", data.score.QrId)
						assert.Equal(t, policy.ScoreType_Result, data.score.Type)
						assert.Equal(t, 100, int(data.score.Value))
						assert.Equal(t, 100, int(data.score.ScoreCompletion))
						assert.Equal(t, 100, int(data.score.DataCompletion))
						assert.Equal(t, 0, int(data.score.DataTotal))
					})
				})

				t.Run("when not isQuery", func(t *testing.T) {
					t.Run("when score", func(t *testing.T) {
						nodeData := newNodeData()

						nodeData.childScores = map[NodeID]*reportingJobResult{
							"rjID1": {
								score: &policy.Score{
									QrId:            "qrid1",
									Type:            policy.ScoreType_Result,
									Value:           100,
									ScoreCompletion: 100,
									DataTotal:       9,
									DataCompletion:  100,
								},
							},
							"rjID2": {
								score: &policy.Score{
									QrId:            "qrid2",
									Type:            policy.ScoreType_Result,
									Value:           0,
									ScoreCompletion: 0,
									DataTotal:       10,
									DataCompletion:  0,
								},
							},
						}
						nodeData.datapoints = map[NodeID]*reportingJobDatapoint{
							"checksum1": {},
						}

						nodeData.initialize()
						data := nodeData.recalculate()

						require.NotNil(t, data)
						nodeData.consume(NodeID("rjID2"), &envelope{
							score: &policy.Score{
								QrId:            "qrid2",
								Type:            policy.ScoreType_Result,
								Value:           100,
								ScoreCompletion: 50,
								DataTotal:       10,
								DataCompletion:  100,
							},
						})
						data = nodeData.recalculate()

						assert.Equal(t, "testqueryid", data.score.QrId)
						assert.Equal(t, policy.ScoreType_Result, data.score.Type)
						assert.Equal(t, 100, int(data.score.Value))
						assert.Equal(t, 75, int(data.score.ScoreCompletion))
						assert.Equal(t, 95, int(data.score.DataCompletion))
						assert.Equal(t, 20, int(data.score.DataTotal))
					})

					t.Run("when score with error", func(t *testing.T) {
						nodeData := newNodeData()

						nodeData.childScores = map[NodeID]*reportingJobResult{
							"rjID1": {
								score: &policy.Score{
									QrId:            "qrid1",
									Type:            policy.ScoreType_Result,
									Value:           100,
									ScoreCompletion: 100,
									DataTotal:       100,
									DataCompletion:  100,
								},
							},
							"rjID2": {
								score: &policy.Score{
									QrId:            "qrid2",
									Type:            policy.ScoreType_Error,
									Value:           50,
									ScoreCompletion: 100,
								},
							},
						}
						nodeData.featureFlagFailErrors = true
						nodeData.initialize()
						data := nodeData.recalculate()

						assert.Equal(t, "testqueryid", data.score.QrId)
						assert.Equal(t, policy.ScoreType_Result, data.score.Type)
						assert.Equal(t, 75, int(data.score.Value))
						assert.Equal(t, 100, int(data.score.ScoreCompletion))
					})

					t.Run("when result", func(t *testing.T) {
						nodeData := newNodeData()

						nodeData.childScores = map[NodeID]*reportingJobResult{
							"rjID1": {
								score: &policy.Score{
									QrId:            "qrid1",
									Type:            policy.ScoreType_Result,
									Value:           100,
									ScoreCompletion: 100,
									DataTotal:       9,
									DataCompletion:  100,
								},
							},
							"rjID2": {
								score: &policy.Score{
									QrId:            "qrid2",
									Type:            policy.ScoreType_Result,
									Value:           100,
									ScoreCompletion: 50,
									DataTotal:       10,
									DataCompletion:  100,
								},
							},
						}
						nodeData.datapoints = map[NodeID]*reportingJobDatapoint{
							"checksum1": {},
						}

						nodeData.initialize()
						data := nodeData.recalculate()

						require.NotNil(t, data)
						nodeData.consume(NodeID("checksum1"), &envelope{
							res: llx.BoolTrue.Result().RawResultV2(),
						})
						data = nodeData.recalculate()

						require.NotNil(t, data)
						require.NotNil(t, data.score)
						assert.Equal(t, "testqueryid", data.score.QrId)
						assert.Equal(t, policy.ScoreType_Result, data.score.Type)
						assert.Equal(t, 100, int(data.score.Value))
						assert.Equal(t, 75, int(data.score.ScoreCompletion))
						assert.Equal(t, 100, int(data.score.DataCompletion))
						assert.Equal(t, 20, int(data.score.DataTotal))
					})
				})
			})
			t.Run("does not recalculate after complete", func(t *testing.T) {
				nodeData := newNodeData()

				nodeData.childScores = map[NodeID]*reportingJobResult{
					"rjID1": {
						score: &policy.Score{
							QrId:            "qrid1",
							Type:            policy.ScoreType_Result,
							Value:           100,
							ScoreCompletion: 100,
							DataTotal:       9,
							DataCompletion:  100,
						},
					},
					"rjID2": {
						score: &policy.Score{
							QrId:            "qrid2",
							Type:            policy.ScoreType_Result,
							Value:           100,
							ScoreCompletion: 50,
							DataTotal:       10,
							DataCompletion:  100,
						},
					},
				}
				nodeData.datapoints = map[NodeID]*reportingJobDatapoint{
					"checksum1": {},
				}

				nodeData.initialize()
				data := nodeData.recalculate()

				require.NotNil(t, data)
				nodeData.consume(NodeID("checksum1"), &envelope{
					res: llx.BoolTrue.Result().RawResultV2(),
				})
				nodeData.consume(NodeID("rjID2"), &envelope{
					score: &policy.Score{
						QrId:            "qrid2",
						Type:            policy.ScoreType_Result,
						Value:           100,
						ScoreCompletion: 100,
						DataTotal:       10,
						DataCompletion:  100,
					},
				})
				data = nodeData.recalculate()

				require.NotNil(t, data)
				require.NotNil(t, data.score)
				assert.Equal(t, "testqueryid", data.score.QrId)
				assert.Equal(t, policy.ScoreType_Result, data.score.Type)
				assert.Equal(t, 100, int(data.score.Value))
				assert.Equal(t, 100, int(data.score.ScoreCompletion))
				assert.Equal(t, 100, int(data.score.DataCompletion))
				assert.Equal(t, 20, int(data.score.DataTotal))

				nodeData.consume(NodeID("rjID2"), &envelope{
					score: &policy.Score{
						QrId:            "qrid2",
						Type:            policy.ScoreType_Result,
						Value:           0,
						ScoreCompletion: 50,
						DataTotal:       10,
						DataCompletion:  100,
					},
				})

				data = nodeData.recalculate()
				assert.Nil(t, data)
			})
		})
	})
}

type progressMock struct {
	f func(current int, total int)
}

func (p *progressMock) Open() error                       { return nil }
func (p *progressMock) OnProgress(current int, total int) { p.f(current, total) }
func (p *progressMock) Score(string)                      {}
func (p *progressMock) Errored()                          {}
func (p *progressMock) NotApplicable()                    {}
func (p *progressMock) Completed()                        {}
func (p *progressMock) Close()                            {}

func TestCollectionFinisherNode(t *testing.T) {
	newNodeData := func(reporter func(current int, total int)) *CollectionFinisherNodeData {
		data := &CollectionFinisherNodeData{
			progressReporter: &progressMock{f: reporter},
			doneChan:         make(chan struct{}),
			assetPlatformId:  "assetPlatformId",
		}
		return data
	}

	results := map[string]*llx.RawResult{
		"codeID1": {
			CodeID: "codeID1",
			Data:   llx.BoolData(true),
		},
	}

	t.Run("initialize/recalculate", func(t *testing.T) {
		t.Run("recalculates if there are no remaining datapoints", func(t *testing.T) {
			nodeData := newNodeData(func(completed int, total int) {
				assert.Equal(t, 0, completed)
				assert.Equal(t, 0, total)
			})
			nodeData.totalDatapoints = 0
			nodeData.remainingDatapoints = map[string]struct{}{}

			nodeData.initialize()
			nodeData.recalculate()

			select {
			case _, ok := <-nodeData.doneChan:
				assert.False(t, ok)
			default:
				assert.Fail(t, "expected channel to be closed")
			}
		})
		t.Run("does not recalculate if there are remaining datapoints", func(t *testing.T) {
			nodeData := newNodeData(func(completed int, total int) {
				assert.Fail(t, "should not recalculate")
			})

			nodeData.totalDatapoints = 2
			nodeData.remainingDatapoints = map[string]struct{}{
				"codeID1": {},
				"codeID2": {},
			}

			nodeData.initialize()
			nodeData.recalculate()

			select {
			case _, _ = <-nodeData.doneChan:
				assert.Fail(t, "expected channel to be open")
			default:
			}
		})
	})

	t.Run("consume/recalculate", func(t *testing.T) {
		t.Run("notifies progress when partially complete", func(t *testing.T) {
			progressCalled := false
			nodeData := newNodeData(func(completed int, total int) {
				progressCalled = true
				assert.Equal(t, 1, completed)
				assert.Equal(t, 2, total)
			})
			nodeData.totalDatapoints = 2
			nodeData.remainingDatapoints = map[string]struct{}{
				"codeID1": {},
				"codeID2": {},
			}
			nodeData.initialize()
			nodeData.consume("codeID1", &envelope{
				res: results["codeID1"],
			})
			nodeData.recalculate()

			assert.True(t, progressCalled)
			select {
			case _, _ = <-nodeData.doneChan:
				assert.Fail(t, "expected channel to be open")
			default:
			}
		})
		t.Run("notifies progress and signals finish when fully complete", func(t *testing.T) {
			progressCalled := false
			nodeData := newNodeData(func(completed int, total int) {
				progressCalled = true
				assert.Equal(t, 1, completed)
				assert.Equal(t, 1, total)
			})
			nodeData.totalDatapoints = 1
			nodeData.remainingDatapoints = map[string]struct{}{
				"codeID1": {},
			}
			nodeData.initialize()
			nodeData.consume("codeID1", &envelope{
				res: results["codeID1"],
			})
			nodeData.recalculate()

			assert.True(t, progressCalled)
			select {
			case _, ok := <-nodeData.doneChan:
				assert.False(t, ok)
			default:
				assert.Fail(t, "expected channel to be closed")
			}
		})
	})
}

func TestDatapointCollectorNode(t *testing.T) {
	newNodeData := func(collectorFunc func(results []*llx.RawResult)) *DatapointCollectorNodeData {
		data := &DatapointCollectorNodeData{
			unreported: make(map[string]*llx.RawResult),
			collectors: []DatapointCollector{
				&FuncCollector{
					SinkDataFunc: collectorFunc,
				},
			},
		}
		return data
	}

	initExpectedData := func() map[string]*llx.RawResult {
		return map[string]*llx.RawResult{
			"codeID1": {
				CodeID: "codeID1",
				Data:   llx.BoolData(true),
			},
			"codeID2": {
				CodeID: "codeID2",
				Data:   llx.BoolData(false),
			},
		}
	}
	t.Run("initialize/recalculate", func(t *testing.T) {
		t.Run("recalculates if unreported datapoints are available", func(t *testing.T) {
			collected := map[string]int{}
			expectedData := initExpectedData()
			nodeData := newNodeData(func(results []*llx.RawResult) {
				for _, r := range results {
					assert.Equal(t, expectedData[r.CodeID], r)
					collected[r.CodeID] = collected[r.CodeID] + 1
				}
			})

			nodeData.unreported = expectedData

			nodeData.initialize()
			nodeData.recalculate()

			assert.Equal(t, 2, len(collected))
			for _, v := range collected {
				assert.Equal(t, 1, v)
			}
		})

		t.Run("does not recalculate if no unreported data", func(t *testing.T) {
			calls := 0
			nodeData := newNodeData(func(results []*llx.RawResult) {
				calls += 1
			})

			nodeData.initialize()
			nodeData.recalculate()

			assert.Equal(t, 0, calls)
		})
	})

	t.Run("consume/recalculate", func(t *testing.T) {
		t.Run("recalculates if unreported datapoints are available", func(t *testing.T) {
			collected := map[string]int{}
			expectedData := initExpectedData()

			nodeData := newNodeData(func(results []*llx.RawResult) {
				for _, r := range results {
					assert.Equal(t, expectedData[r.CodeID], r)
					collected[r.CodeID] = collected[r.CodeID] + 1
				}
			})

			nodeData.initialize()
			nodeData.consume("codeID1", &envelope{
				res: expectedData["codeID1"],
			})
			nodeData.consume("rjID1", &envelope{
				res: expectedData["codeID2"],
			})
			nodeData.recalculate()

			assert.Equal(t, 2, len(collected))
			for _, v := range collected {
				assert.Equal(t, 1, v)
			}
		})
	})
}

func TestScoreCollectorNode(t *testing.T) {
	newNodeData := func(collectorFunc func(scores []*policy.Score)) *ScoreCollectorNodeData {
		data := &ScoreCollectorNodeData{
			unreported: make(map[string]*policy.Score),
			collectors: []ScoreCollector{
				&FuncCollector{
					SinkScoreFunc: collectorFunc,
				},
			},
		}
		return data
	}

	initExpectedScores := func() map[string]*policy.Score {
		return map[string]*policy.Score{
			"queryID1": {
				QrId:            "queryID1",
				Type:            policy.ScoreType_Result,
				Value:           55,
				ScoreCompletion: 100,
			},
			"rjID1": {
				QrId:            "rjID1",
				Type:            policy.ScoreType_Result,
				Value:           75,
				ScoreCompletion: 100,
				DataTotal:       1,
				DataCompletion:  100,
			},
		}
	}

	t.Run("initialize/recalculate", func(t *testing.T) {
		t.Run("recalculates if unreported scores are available", func(t *testing.T) {
			collected := map[string]int{}
			expectedScores := initExpectedScores()
			nodeData := newNodeData(func(scores []*policy.Score) {
				for _, s := range scores {
					assert.Equal(t, expectedScores[s.QrId], s)
					collected[s.QrId] = collected[s.QrId] + 1
				}
			})

			nodeData.unreported = expectedScores

			nodeData.initialize()
			nodeData.recalculate()

			assert.Equal(t, 2, len(collected))
			for _, v := range collected {
				assert.Equal(t, 1, v)
			}
		})

		t.Run("does not recalculate if no unreported scores", func(t *testing.T) {
			calls := 0
			nodeData := newNodeData(func(scores []*policy.Score) {
				calls += 1
			})

			nodeData.initialize()
			nodeData.recalculate()

			assert.Equal(t, 0, calls)
		})
	})

	t.Run("consume/recalculate", func(t *testing.T) {
		t.Run("recalculates if unreported scores are available", func(t *testing.T) {
			collected := map[string]int{}
			expectedScores := initExpectedScores()

			nodeData := newNodeData(func(scores []*policy.Score) {
				for _, s := range scores {
					assert.Equal(t, expectedScores[s.QrId], s)
					collected[s.QrId] = collected[s.QrId] + 1
				}
			})

			nodeData.initialize()
			nodeData.consume("queryID1", &envelope{
				score: expectedScores["queryID1"],
			})
			nodeData.consume("rjID1", &envelope{
				score: expectedScores["rjID1"],
			})
			nodeData.recalculate()

			assert.Equal(t, 2, len(collected))
			for _, v := range collected {
				assert.Equal(t, 1, v)
			}
		})
	})
}
