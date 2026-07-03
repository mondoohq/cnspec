ifndef LATEST_VERSION_TAG
# echo "read LATEST_VERSION_TAG from git"
LATEST_VERSION_TAG=$(shell git describe --abbrev=0 --tags)
endif

ifndef MANIFEST_VERSION
# echo "read MANIFEST_VERSION from git"
MANIFEST_VERSION=$(shell git describe --abbrev=0 --tags)
endif

ifndef VERSION
# echo "read VERSION from git"
VERSION=${LATEST_VERSION_TAG}+$(shell git rev-list --count HEAD)
endif
MAJOR_VERSION=v13

# use LDFLAGSEXTRA to pass additional ldflags to the build
LDFLAGS="-s -w -X go.mondoo.com/mql/${MAJOR_VERSION}.Version=${LATEST_VERSION_TAG} -X go.mondoo.com/cnspec/${MAJOR_VERSION}.Version=${LATEST_VERSION_TAG} ${LDFLAGSEXTRA}"
LDFLAGSDIST=-tags production -ldflags ${LDFLAGS}

.PHONY: info/ldflags
info/ldflags:
	$(info go run -ldflags ${LDFLAGS} apps/cnspec/cnspec.go)
	@:

#   🧹 CLEAN   #

clean/proto:
	find . -not -path './.*' \( -name '*.ranger.go' -or -name '*.pb.go' -or -name '*.actions.go' -or -name '*-packr.go' -or -name '*.swagger.json' \) -delete

.PHONY: version
version:
	@echo $(VERSION)


#   🔨 TOOLS       #

prep: prep/tools

# we need mql due to a few proto files requiring it. proto doesn't resolve dependencies for us
# or download them from the internet, so we are making sure the repo exists this way.
# An alternative (especially for local development) is to soft-link a local copy of the repo
# yourself. We don't pin submodules at this time, but we may want to check if they are up to date here.
prep/repos:
	test -x mql || git clone https://github.com/mondoohq/mql.git mql

prep/repos/update: prep/repos
	cd mql; git checkout main && git pull; cd -;

prep/tools/windows:
	go get google.golang.org/protobuf
	go get -u gotest.tools/gotestsum

prep/tools:
	# additional helper
	go install gotest.tools/gotestsum@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest


#   🌙 cnspec   #

cnspec/generate: clean/proto cli/generate policy/generate reporter/generate

.PHONY: cli
cli/generate:
	go generate ./cli/reporter

.PHONY: policy
policy/generate:
	go generate ./policy
	go generate ./policy/scan
	go generate ./policy/scandb
	go generate ./internal/sbom
	go generate ./internal/bundle/yacit

reporter/generate:
	go generate ./cli/reporter

#   🏗 Binary   #

.PHONY: cnspec/build
cnspec/build:
	go build -o cnspec ${LDFLAGSDIST} apps/cnspec/cnspec.go

.PHONY: cnspec/build/linux
cnspec/build/linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGSDIST} apps/cnspec/cnspec.go

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

test/lint/content:
	cnspec policy lint ./content
	cnspec policy lint ./content/querypacks

test: test/go test/lint

benchmark/go:
	go test -bench=. -benchmem go.mondoo.com/cnspec/${MAJOR_VERSION}/policy/scan/benchmark

test/go: cnspec/generate test/go/plain

test/go/plain:
	go test -cover $(shell go list ./...)

test/go/plain-ci: prep/tools
	gotestsum --junitfile report.xml --format pkgname -- -cover $(shell go list ./... | grep -v '/vendor/')

# Content IaC-variant suites (Terraform / CloudFormation / Bicep / Dockerfile /
# Kubernetes) validate every policy check against its per-check pass/fail fixtures
# in content/iac-variant-testdata. They are isolated behind the `iac_variants` build
# tag so they never run in the default `go test ./...` (they download extra
# providers and run many provider-backed scans). Concurrency is kept conservative
# to avoid provider-subprocess contention; override with IAC_VARIANT_PARALLEL.
IAC_VARIANT_PARALLEL ?= 4

.PHONY: test/go/content-iac test/go/content-iac/terraform test/go/content-iac/cloudformation test/go/content-iac/bicep test/go/content-iac/dockerfile test/go/content-iac/kubernetes
test/go/content-iac: prep/tools
	go test -tags iac_variants -parallel $(IAC_VARIANT_PARALLEL) -run 'TestTerraformVariants|TestCloudFormationVariants|TestBicepVariants|TestDockerfileVariants|TestKubernetesManifestVariants' ./content

test/go/content-iac/terraform: prep/tools
	go test -tags iac_variants -parallel $(IAC_VARIANT_PARALLEL) -run '^TestTerraformVariants$$' ./content

test/go/content-iac/cloudformation: prep/tools
	go test -tags iac_variants -parallel $(IAC_VARIANT_PARALLEL) -run '^TestCloudFormationVariants$$' ./content

test/go/content-iac/bicep: prep/tools
	go test -tags iac_variants -parallel $(IAC_VARIANT_PARALLEL) -run '^TestBicepVariants$$' ./content

test/go/content-iac/dockerfile: prep/tools
	go test -tags iac_variants -parallel $(IAC_VARIANT_PARALLEL) -run '^TestDockerfileVariants$$' ./content

test/go/content-iac/kubernetes: prep/tools
	go test -tags iac_variants -parallel $(IAC_VARIANT_PARALLEL) -run '^TestKubernetesManifestVariants$$' ./content

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
	golangci-lint run --timeout 10m --config .github/.golangci.yaml --new-from-rev $(shell git log -n 1 origin/main --pretty=format:"%H")

.PHONY: skills/generate
skills/generate:
	go run ./scripts/generate-agents

.PHONY: skills/generate/check
skills/generate/check:
	go run ./scripts/generate-agents --check

.PHONY: install/skills
install/skills:
	@echo "Installing cnspec skills to ~/.claude ..."
	@for skill_dir in skills/*/; do \
		[ -f "$$skill_dir/SKILL.md" ] || continue; \
		name=$$(basename $$skill_dir); \
		mkdir -p ~/.claude/skills/$$name; \
		cp $$skill_dir/*.md ~/.claude/skills/$$name/ 2>/dev/null || true; \
		if [ -d "$$skill_dir/references" ]; then \
			mkdir -p ~/.claude/skills/$$name/references; \
			cp $$skill_dir/references/*.md ~/.claude/skills/$$name/references/ 2>/dev/null || true; \
		fi; \
		if [ -d "$$skill_dir/samples" ]; then \
			mkdir -p ~/.claude/skills/$$name/samples; \
			cp $$skill_dir/samples/*.md ~/.claude/skills/$$name/samples/ 2>/dev/null || true; \
		fi; \
		echo "  ✓ $$name"; \
	done
	@for cmd in skills/*/commands/*.md; do \
		[ -f "$$cmd" ] || continue; \
		mkdir -p ~/.claude/commands; \
		cp $$cmd ~/.claude/commands/; \
		echo "  ✓ command: $$(basename $$cmd)"; \
	done
	@echo "Done. Skills available in all projects."

license: license/headers/check

license/headers/check:
	copywrite headers --plan

license/headers/apply:
	copywrite headers

#   📈 METRICS       #

metrics/start: metrics/grafana/start metrics/prometheus/start

metrics/prometheus/start:
	APP_NAME=cnspec VERSION=${VERSION} prometheus --config.file=prometheus.yml

metrics/grafana/start:
	docker run -d --name=grafana \
		-p 3000:3000               \
		grafana/grafana

metrics/grafana/stop:
	docker stop grafana
