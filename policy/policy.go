package policy

//go:generate protoc --proto_path=../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. --iam-actions_out=. --rangerrpc-swagger_out=. policy.proto
