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

LDFLAGS=-ldflags "-s -w -X go.mondoo.com/cnspec/v11.Version=${VERSION} -X go.mondoo.com/cnspec/v11.Build=${TAG}" # -linkmode external -extldflags=-static
LDFLAGSDIST=-tags production -ldflags "-s -w -X go.mondoo.com/cnquery/v11.Version=${LATEST_VERSION_TAG} -X go.mondoo.com/cnquery/v11.Build=${TAG} -X go.mondoo.com/cnspec/v11.Version=${LATEST_VERSION_TAG} -X go.mondoo.com/cnspec/v11.Build=${TAG} -s -w"

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

# we need cnquery due to a few proto files requiring it. proto doesn't resolve dependencies for us
# or download them from the internet, so we are making sure the repo exists this way.
# An alternative (especially for local development) is to soft-link a local copy of the repo
# yourself. We don't pin submodules at this time, but we may want to check if they are up to date here.
prep/repos:
	test -x cnquery || git clone https://github.com/mondoohq/cnquery.git

prep/repos/update: prep/repos
	cd cnquery; git checkout main && git pull; cd -;

prep/tools/windows:
	go get google.golang.org/protobuf
	go get -u gotest.tools/gotestsum

prep/tools:
	# protobuf tooling
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install go.mondoo.com/ranger-rpc/protoc-gen-rangerrpc@latest
	go install go.mondoo.com/ranger-rpc/protoc-gen-rangerrpc-swagger@latest
	# additional helper
	go install gotest.tools/gotestsum@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest


#   🌙 cnspec   #

cnspec/generate: clean/proto cli/generate policy/generate reporter/generate

.PHONY: cli
cli/generate:
	go generate ./cli/reporter

.PHONY: policy
policy/generate:
	go generate ./policy
	go generate ./policy/scan
	go generate ./internal/bundle/yacit

reporter/generate:
	go generate ./cli/reporter

#   🏗 Binary   #

.PHONY: cnspec/build
cnspec/build:
	go build -o cnspec ${LDFLAGSDIST} apps/cnspec/cnspec.go

.PHONY: cnspec/build/linux
cnspec/build/linux:
	GOOS=linux GOARCH=amd64 go build ${LDFLAGSDIST} apps/cnspec/cnspec.go

.PHONY: cnspec/build/linux/arm
cnspec/build/linux/arm:
	GOOS=linux GOARCH=arm64 go build ${LDFLAGSDIST} apps/cnspec/cnspec.go

.PHONY: cnspec/build/windows
cnspec/build/windows:
	GOOS=windows GOARCH=amd64 go build ${LDFLAGSDIST} apps/cnspec/cnspec.go

.PHONY: cnspec/install
cnspec/install:
	GOBIN=${GOPATH}/bin go install ${LDFLAGSDIST} apps/cnspec/cnspec.go

cnspec/dist/goreleaser/stable:
	goreleaser release --clean --skip=validate,publish -f .goreleaser.yml --timeout 120m

cnspec/dist/goreleaser/edge:
	goreleaser release --clean --skip=validate,publish -f .goreleaser.yml --timeout 120m --snapshot


#   ⛹🏽‍ Testing   #

test/lint: test/lint/golangci-lint/run

test: test/go test/lint

benchmark/go:
	go test -bench=. -benchmem go.mondoo.com/cnspec/v11/policy/scan/benchmark

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

.PHONY: test/lint/golangci-lint/run/new
test/lint/golangci-lint/run/new: prep/tools
	golangci-lint --version
	golangci-lint run --timeout 10m --config .github/.golangci.yml --new-from-rev $(shell git log -n 1 origin/main --pretty=format:"%H")

license: license/headers/check

license/headers/check:
	copywrite headers --plan

license/headers/apply:
	copywrite headers

#   📈 METRICS       #

metrics/start: metrics/grafana/start metrics/prometheus/start

metrics/prometheus/start:
	APP_NAME=cnspec VERSION=${VERSION} BUILD=${TAG} prometheus --config.file=prometheus.yml

metrics/grafana/start:
	docker run -d --name=grafana \
		-p 3000:3000               \
		grafana/grafana

metrics/grafana/stop:
	docker stop grafana
