// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tfgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

func TestConvertTypeToCty(t *testing.T) {
	t.Run("success_string", func(t *testing.T) {
		val := "hello"
		result, err := convertTypeToCty(val)
		assert.NoError(t, err)
		assert.Equal(t, cty.String, result.Type())
		assert.Equal(t, cty.StringVal(val), result)
	})

	t.Run("success_int", func(t *testing.T) {
		val := 42
		result, err := convertTypeToCty(val)
		assert.NoError(t, err)
		assert.Equal(t, cty.Number, result.Type())
		assert.Equal(t, cty.NumberIntVal(int64(val)), result)
	})

	t.Run("success_bool", func(t *testing.T) {
		val := true
		result, err := convertTypeToCty(val)
		assert.NoError(t, err)
		assert.Equal(t, cty.Bool, result.Type())
		assert.Equal(t, cty.BoolVal(val), result)
	})

	t.Run("success_slice_of_strings", func(t *testing.T) {
		val := []string{"apple", "banana", "cherry"}
		result, err := convertTypeToCty(val)
		assert.NoError(t, err)
		expected := cty.ListVal([]cty.Value{
			cty.StringVal("apple"),
			cty.StringVal("banana"),
			cty.StringVal("cherry"),
		})
		assert.Equal(t, cty.List(cty.String), result.Type())
		assert.Equal(t, expected, result)
	})

	t.Run("success_map_of_strings", func(t *testing.T) {
		val := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}
		result, err := convertTypeToCty(val)
		assert.NoError(t, err)
		expected := cty.MapVal(map[string]cty.Value{
			"key1": cty.StringVal("value1"),
			"key2": cty.StringVal("value2"),
		})
		assert.Equal(t, cty.Map(cty.String), result.Type())
		assert.Equal(t, expected, result)
	})

	t.Run("success_empty_string", func(t *testing.T) {
		val := ""
		result, err := convertTypeToCty(val)
		assert.NoError(t, err)
		assert.Equal(t, cty.String, result.Type())
		assert.Equal(t, cty.StringVal(val), result)
	})

	t.Run("success_nil_value", func(t *testing.T) {
		var val any
		result, err := convertTypeToCty(val)
		assert.Error(t, err)
		assert.Equal(t, cty.NilVal, result)
	})

	t.Run("error_unsupported_type", func(t *testing.T) {
		val := struct{}{} // Unsupported type
		result, err := convertTypeToCty(val)
		assert.Error(t, err)
		assert.Equal(t, cty.NilVal, result)
	})

	t.Run("success_empty_slice", func(t *testing.T) {
		val := []string{}
		result, err := convertTypeToCty(val)
		assert.NoError(t, err)
		assert.Equal(t, cty.List(cty.String), result.Type())
		assert.Equal(t, cty.ListValEmpty(cty.String), result)
	})

	t.Run("success_empty_map", func(t *testing.T) {
		val := map[string]string{}
		result, err := convertTypeToCty(val)
		assert.NoError(t, err)
		assert.Equal(t, cty.NilVal, result)
	})

	t.Run("success_large_int", func(t *testing.T) {
		val := int64(9223372036854775807) // Max int64 value
		result, err := convertTypeToCty(val)
		assert.NoError(t, err)
		assert.Equal(t, cty.NumberIntVal(val), result)
	})
}
