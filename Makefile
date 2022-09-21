ifndef LATEST_VERSION_TAG
# echo "read LATEST_VERSION_TAG from git"
LATEST_VERSION_TAG=$(shell git describe --abbrev=0 --tags)
endif

ifndef MANIFEST_VERSION
# echo "read MANIFEST_VERSION from git"
MANIFEST_VERSION=$(shell git describe --abbrev=0 --tags)
endif

ifndef TAG
# echo "read TAG from git"
TAG=$(shell git log --pretty=format:'%h' -n 1)
endif

ifndef VERSION
# echo "read VERSION from git"
VERSION=${LATEST_VERSION_TAG}+$(shell git rev-list --count HEAD)
endif

LDFLAGS=-ldflags "-s -w -X go.mondoo.com/cnspec.Version=${VERSION} -X go.mondoo.com/cnspec.Build=${TAG}" # -linkmode external -extldflags=-static
LDFLAGSDIST=-tags production -ldflags "-s -w -X go.mondoo.com/cnspec.Version=${LATEST_VERSION_TAG} -X go.mondoo.com/cnspec.Build=${TAG} -s -w"

.PHONY: info/ldflags
info/ldflags:
	$(info go run ${LDFLAGS} apps/cnspec/cnspec.go)
	@:

#   🧹 CLEAN   #

clean/proto:
	find . -not -path './.*' \( -name '*.ranger.go' -or -name '*.pb.go' -or -name '*.actions.go' -or -name '*-packr.go' -or -name '*.swagger.json' \) -delete

.PHONY: version
version:
	@echo $(VERSION)


#   🔨 TOOLS       #

prep: prep/tools

# TODO: we require cnquery to be there!

prep/tools/windows:
	go get github.com/golang/protobuf/proto
	go get -u gotest.tools/gotestsum

prep/tools:
	# protobuf tooling
	command -v protoc-gen-go || go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	command -v protoc-gen-rangerrpc || go install go.mondoo.com/ranger-rpc/protoc-gen-rangerrpc@latest
	command -v protoc-gen-rangerrpc-swagger || go install go.mondoo.com/ranger-rpc/protoc-gen-rangerrpc-swagger@latest
	# additional helper
	command -v gotestsum || go install gotest.tools/gotestsum@latest
	command -v golangci-lint || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest


#   🌙 cnspec   #

cnspec/generate: clean/proto cli/generate policy/generate

.PHONY: cli
cli/generate:
	go generate ./cli/reporter

.PHONY: policy
policy/generate:
	go generate ./policy

#   🏗 Binary   #

.PHONY: cnspec/install
cnspec/install:
	GOBIN=${GOPATH}/bin go install ${LDFLAGSDIST} apps/cnspec/cnspec.go

#   ⛹🏽‍ Testing   #

test/lint: test/lint/golangci-lint/run

test: test/go test/lint

test/go: cnspec/generate test/go/plain

test/go/plain:
	# TODO /motor/docker/docker_engine cannot be executed inside of docker
	go test -cover $(shell go list ./... | grep -v '/motor/discovery/docker_engine')

test/go/plain-ci: prep/tools
	gotestsum --junitfile report.xml --format pkgname -- -cover $(shell go list ./... | grep -v '/vendor/' | grep -v '/motor/discovery/docker_engine')

.PHONY: test/lint/staticcheck
test/lint/staticcheck:
	staticcheck $(shell go list ./... | grep -v /ent/ | grep -v /benchmark/)

.PHONY: test/lint/govet
test/lint/govet:
	go vet $(shell go list ./... | grep -v /ent/ | grep -v /benchmark/)

.PHONY: test/lint/golangci-lint/run
test/lint/golangci-lint/run: prep/tools
	golangci-lint --version
	golangci-lint run
