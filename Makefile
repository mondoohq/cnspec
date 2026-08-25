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

# use LDFLAGSEXTRA to pass additional ldflags to the build
LDFLAGS="-s -w -X go.mondoo.com/mql.Version=${LATEST_VERSION_TAG} -X go.mondoo.com/cnspec.Version=${LATEST_VERSION_TAG} ${LDFLAGSEXTRA}"
LDFLAGSDIST=-tags production -ldflags ${LDFLAGS}

# Windows version resource generator. The PE version fields are numeric, so strip
# the leading "v" from the tag (v13.32.1 -> 13.32.1) to match release builds,
# which use goreleaser's already-stripped {{ .Version }}.
GO_WINRES=github.com/tc-hib/go-winres@v0.3.3
WINRES_VERSION=$(LATEST_VERSION_TAG:v%=%)

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
# Which mql line code generation reads. cnspec's *.pb.go bake in the
# go_package of mql's protos, so this has to be the mql branch this cnspec
# branch imports, or generation emits imports go.mod does not provide.
# main tracks mql main. A support branch cut from here has to set this to
# its own major's mql branch: mql no longer carries the major in its module
# path (mondoohq/mql#10234), so nothing in the tree can infer it.
MQL_BRANCH ?= main

prep/repos:
	test -x mql || git clone -b $(MQL_BRANCH) https://github.com/mondoohq/mql.git mql

prep/repos/update: prep/repos
	cd mql; git checkout $(MQL_BRANCH) && git pull; cd -;

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

# Generates the Windows version resource (VERSIONINFO) and application manifest
# into apps/cnspec/rsrc_windows_<arch>.syso. Release builds run the equivalent
# step from the `before` hook in .goreleaser.yml.
.PHONY: cnspec/winres
cnspec/winres:
	go run ${GO_WINRES} make --in apps/cnspec/winres/winres.json --out apps/cnspec/rsrc \
		--arch amd64,arm64 \
		--file-version ${WINRES_VERSION} --product-version ${WINRES_VERSION}

# NOTE: the build target must stay a package path, not apps/cnspec/cnspec.go. Go
# only links rsrc_windows_*.syso when building a package directory; passing an
# explicit .go file silently drops the version resource and manifest.
.PHONY: cnspec/build/windows
cnspec/build/windows: cnspec/winres
	GOOS=windows GOARCH=amd64 go build ${LDFLAGSDIST} ./apps/cnspec

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
	go test -bench=. -benchmem go.mondoo.com/cnspec/policy/scan/benchmark

test/go: cnspec/generate test/go/plain

test/go/plain:
	go test -cover $(shell go list ./...)

test/go/plain-ci: prep/tools
	gotestsum --junitfile report.xml --format pkgname -- -cover $(shell go list ./... | grep -v '/vendor/')

#   📋 Content validation   #
#
# Every check that runs against the policies in content/. All of it is
# documented in content/validation/README.md — what each one proves, when CI
# runs it, and how to scope a run down to a single check.

VALIDATION := content/validation

.PHONY: test/content test/content/lint test/content/spelling test/content/scans test/content/compliance
.PHONY: test/content/iac test/content/iac/terraform test/content/iac/cloudformation test/content/iac/bicep
.PHONY: test/content/iac/dockerfile test/content/iac/kubernetes test/content/iac/coverage test/content/iac/remediation
.PHONY: test/content/remediation test/content/remediation/terraform test/content/remediation/cloudformation
.PHONY: test/content/remediation/bicep test/content/remediation/ansible test/content/remediation/powershell
.PHONY: test/content/remediation/bash test/content/remediation/chef
.PHONY: test/content/commands test/content/upstream test/content/upstream/unit

# The checks that run with nothing installed but Go and cnspec. Two groups are
# deliberately left out, each for its own reason:
#
#   test/content/iac         downloads providers and runs thousands of scans
#   test/content/remediation each validator needs its language's linter, and
#   test/content/commands    each cloud needs that cloud's CLI on PATH
#
# CI runs all of them; locally, run the one that covers what you touched.
test/content: test/content/lint test/content/scans test/content/compliance

# Structure, MQL compilation, and schema. Catches a check that cannot compile
# before any suite tries to run it.
test/content/lint:
	cnspec policy lint ./content
	cnspec policy lint ./content/querypacks

# Spelling across the repo, using the allowlist in typos.toml. CI runs this via
# the crate-ci/typos action; locally it needs `brew install typos-cli`.
test/content/spelling:
	typos

# Whole-bundle smoke scans: a sample project per policy that should score 100
# or 0. Untagged, so it also runs as part of `make test/go`.
test/content/scans:
	go test -timeout 20m ./$(VALIDATION)/scans

# Compliance-tag mapping invariants. Reads the bundles only, runs no scans.
test/content/compliance:
	go test ./$(VALIDATION)/compliance

# Content IaC-variant suites (Terraform / CloudFormation / Bicep / Dockerfile /
# Kubernetes) validate every policy check against its per-check pass/fail fixtures
# in content/validation/scans/fixtures/iac-variants. They are isolated behind the
# `iac_variants` build tag so they never run in the default `go test ./...` (they
# download extra providers and run many provider-backed scans). Concurrency is kept
# conservative to avoid provider-subprocess contention; override with
# IAC_VARIANT_PARALLEL.
#
# To fan a suite out across parallel CI runners, set IAC_SHARD_TOTAL to the
# runner count and IAC_SHARD_INDEX to each runner's 0-based slot; the harness
# then runs only the scenarios that hash into that shard. Unset means run
# everything. Every suite supports it. See .github/workflows/content-iac-tests.yaml.
IAC_VARIANT_PARALLEL ?= 4

# How long a suite may run before the test binary gives up. The value that
# matters is not "long enough for the scans" -- it is "short enough to be a
# useful failure". The coordinator lock ordering documented in
# content/validation/scans/main_test.go deadlocks rather than errors, and a
# deadlocked run does not fail, it stops; the timeout is what turns that into a
# goroutine dump naming the two blocked stacks.
#
# The default is sized for an unsharded local run of the largest suite. CI runs
# one shard per job and overrides it to something well under the workflow's
# timeout-minutes, so a stuck shard dumps its goroutines (which the job timeout
# would not) and still ends in minutes rather than an hour.
IAC_TIMEOUT ?= 60m

# These targets run plain `go test`; they deliberately do not depend on
# prep/tools, which installs gotestsum and golangci-lint at @latest. Neither is
# used here, and paying two module-proxy resolutions on every one of the ~28
# shard jobs bought nothing.
IAC_TEST := go test -tags iac_variants -parallel $(IAC_VARIANT_PARALLEL) -timeout $(IAC_TIMEOUT) ./$(VALIDATION)/scans

test/content/iac:
	$(IAC_TEST) -run 'TestTerraformVariants|TestCloudFormationVariants|TestBicepVariants|TestDockerfileVariants|TestKubernetesManifestVariants'

test/content/iac/terraform:
	$(IAC_TEST) -run '^TestTerraformVariants$$'

test/content/iac/cloudformation:
	$(IAC_TEST) -run '^TestCloudFormationVariants$$'

test/content/iac/bicep:
	$(IAC_TEST) -run '^TestBicepVariants$$'

test/content/iac/dockerfile:
	$(IAC_TEST) -run '^TestDockerfileVariants$$'

test/content/iac/kubernetes:
	$(IAC_TEST) -run '^TestKubernetesManifestVariants$$'

# Closed loop: scans each IaC variant's own remediation snippet and requires the
# check that recommends it to pass. Sharded like every other suite, since it runs
# one provider-backed scan per variant.
test/content/iac/remediation:
	$(IAC_TEST) -run '^TestRemediationSatisfiesCheck$$'

# Fixture-coverage gate. Runs no scans: it compares the IaC variants declared in
# each policy against the fixtures on disk and fails if any variant lacks a pass
# or a fail fixture. Coverage is at 100%, so this holds it there.
test/content/iac/coverage:
	$(IAC_TEST) -v -run '^TestTerraformVariantCoverage$$'

# Remediation code blocks, linted in their own language. Each target needs that
# language's linter installed; see content/validation/README.md.
test/content/remediation: test/content/remediation/terraform test/content/remediation/cloudformation \
	test/content/remediation/bicep test/content/remediation/ansible test/content/remediation/powershell \
	test/content/remediation/bash test/content/remediation/chef

test/content/remediation/terraform:
	python3 $(VALIDATION)/remediation/code/terraform.py

test/content/remediation/cloudformation:
	python3 $(VALIDATION)/remediation/code/cloudformation.py

test/content/remediation/bicep:
	python3 $(VALIDATION)/remediation/code/bicep.py

test/content/remediation/ansible:
	python3 $(VALIDATION)/remediation/code/ansible.py

test/content/remediation/powershell:
	python3 $(VALIDATION)/remediation/code/powershell.py

test/content/remediation/bash:
	python3 $(VALIDATION)/remediation/code/bash.py

test/content/remediation/chef:
	python3 $(VALIDATION)/remediation/code/chef.py

# Remediation CLI and REST API calls, checked against checked-in grammars and
# OpenAPI specs. Pass CLOUD=aws (or any name the validator lists) to scope it.
CLOUD ?= all
test/content/commands:
	python3 $(VALIDATION)/remediation/commands/validate.py $(CLOUD)

# Reports which upstreams the validators are pinned behind. Hits the network;
# reports only, never fails the build.
test/content/upstream:
	python3 $(VALIDATION)/upstream/check.py --format markdown

# The resolvers behind that report, against recorded upstream payloads. Offline
# and fast, and the one part of this that does fail the build: a resolver that
# reads an index wrongly opens a pull request bumping a pin to a version that
# does not exist.
test/content/upstream/unit:
	python3 -m unittest discover -s $(VALIDATION)/upstream -p "*_test.py"

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
