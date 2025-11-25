// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

//go:generate protoc --plugin=protoc-gen-go=../../scripts/protoc/protoc-gen-go --proto_path=../:../../cnquery:. --go_out=. --go_opt=paths=source_relative cnspec_report.proto
