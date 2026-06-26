#!/usr/bin/env bash
#
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
set -euo pipefail

# Installs the protoc compiler into a directory scoped to this CI run and
# prepends its bin/ to PATH (via $GITHUB_PATH) for subsequent steps.
#
# Replaces arduino/setup-protoc. The codegen plugins (protoc-gen-go,
# protoc-gen-rangerrpc, ...) are Go tools wrapped in scripts/protoc/ and are
# NOT installed here -- only the protoc binary itself. Supports the linux,
# macOS and Windows runners protoc publishes assets for.
#
# The downloaded archive is verified against the SHA-256 digest GitHub records
# for the asset before it is unpacked, so a corrupted or swapped download fails
# the build instead of landing protoc on PATH.
#
# Inputs (env):
#   PROTOC_VERSION  Version spec, e.g. "33.x" (latest 33.*) or "33.6" (exact).
#                   Required; the workflow/composite action passes the
#                   protoc-version from .github/env. The script hard-fails if
#                   it is unset.
#   PROTOC_HOME     Install prefix. Defaults to a per-run dir under
#                   $RUNNER_TEMP so installs never touch /usr/local and don't
#                   leak between runs on a reused self-hosted runner.
#   GH_TOKEN        Optional GitHub token for the releases API (avoids
#                   unauthenticated rate limits). Honored by `gh` automatically.

PROTOC_VERSION="${PROTOC_VERSION:?PROTOC_VERSION is required (e.g. 33.x)}"
PROTOC_HOME="${PROTOC_HOME:-${RUNNER_TEMP:-/tmp}/protoc}"

# Validate the version spec up front. The value is interpolated into a release
# tag, an asset filename, and a jq filter; constraining it to a dotted version
# or a "<major>.x" wildcard means nothing downstream can carry shell, glob or
# jq metacharacters. A dot is required (a bare major like "33" is rejected here
# rather than later, where it would fail the post-resolution check).
if [[ ! "$PROTOC_VERSION" =~ ^[0-9]+\.([0-9]+(\.[0-9]+)*|x)$ ]]; then
  echo "error: invalid PROTOC_VERSION '${PROTOC_VERSION}' (expected e.g. 33.x or 33.6)" >&2
  exit 1
fi

# Resolve a "33.x" wildcard to the latest matching v33.* release tag. An exact
# version (no trailing .x) is used verbatim.
resolve_version() {
  local spec="$1"
  if [[ "$spec" != *.x ]]; then
    echo "$spec"
    return
  fi
  local prefix="${spec%.x}." # e.g. "33."
  # gh paginates releases newest-first; grep -m1 stops at the first matching
  # tag and closes the pipe, so gh short-circuits instead of fetching every
  # page (protobuf has 150+ releases). select(.prerelease | not) drops
  # release-candidate tags (e.g. v33.7-rc1) that would otherwise sort above
  # the latest stable release and be matched first.
  local tag
  tag="$(gh api --paginate repos/protocolbuffers/protobuf/releases \
    --jq '.[] | select(.prerelease | not) | .tag_name' \
    | grep -m1 -E "^v${prefix//./\\.}")"
  if [[ -z "$tag" ]]; then
    echo "error: no protoc release matching ${spec}" >&2
    exit 1
  fi
  echo "${tag#v}" # strip leading "v" -> bare version like 33.6
}

# Resolve the host's protoc release asset name and the path of the unpacked
# binary. protoc names linux/macOS assets "<os>-<arch>" but ships a combined
# "win64"/"win32" asset on Windows, where the binary is protoc.exe. CI runs on
# linux today; macOS and Windows are derived so the cgo build matrix works too.
detect_platform() {
  local m s
  m="$(uname -m)"
  s="$(uname -s)"
  case "$m" in
    x86_64 | amd64) arch="x86_64" ;;
    aarch64 | arm64) arch="aarch_64" ;;
    *)
      echo "error: unsupported architecture ${m}" >&2
      exit 1
      ;;
  esac
  case "$s" in
    Linux) os="linux" ;;
    Darwin) os="osx" ;;
    MINGW* | MSYS* | CYGWIN*) os="win" ;;
    *)
      echo "error: unsupported OS ${s}" >&2
      exit 1
      ;;
  esac
}

# Print the hex SHA-256 of a file, using whichever tool the runner ships.
sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

version="$(resolve_version "$PROTOC_VERSION")"
# A resolved version is always a concrete dotted release (e.g. 33.6); re-check
# it before it is interpolated, in case the releases API returns anything odd.
if [[ ! "$version" =~ ^[0-9]+(\.[0-9]+)+$ ]]; then
  echo "error: resolved unexpected protoc version '${version}'" >&2
  exit 1
fi

detect_platform
if [[ "$os" == "win" ]]; then
  # protoc ships a single combined win64 asset, not "<os>-<arch>", and
  # publishes no Windows ARM build -- fail loudly rather than installing the
  # 32-bit x86 win32 asset on an arm64 runner.
  if [[ "$arch" != "x86_64" ]]; then
    echo "error: protoc does not publish a Windows ${arch} asset" >&2
    exit 1
  fi
  asset="protoc-${version}-win64.zip"
  protoc_bin="${PROTOC_HOME}/bin/protoc.exe"
else
  asset="protoc-${version}-${os}-${arch}.zip"
  protoc_bin="${PROTOC_HOME}/bin/protoc"
fi

# Idempotency: a reused runner may already have this exact version cached.
# With the default per-run PROTOC_HOME (under $RUNNER_TEMP) a cache hit can only
# be a binary this run already downloaded and verified. Overriding PROTOC_HOME
# to a path that persists across runs trusts that cached binary without
# re-verifying its checksum here. -Fqw matches the version as a literal word so
# the dots aren't treated as regex wildcards.
if [[ -x "$protoc_bin" ]] \
  && "$protoc_bin" --version 2>/dev/null | grep -Fqw "$version"; then
  echo "protoc ${version} already present at ${PROTOC_HOME}"
else
  echo "Installing protoc ${version} (${os}-${arch}) into ${PROTOC_HOME}"
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT
  gh release download "v${version}" \
    --repo protocolbuffers/protobuf \
    --pattern "$asset" \
    --dir "$tmp"

  # Verify the archive against the digest GitHub records for this asset before
  # unpacking. Fail closed if the digest is missing or malformed.
  #
  # .digest is GitHub-computed for assets uploaded after the feature shipped
  # (~2025); it is populated for the protoc 33.x releases this repo pins but is
  # null on much older releases (e.g. v28.0). protobuf does NOT publish per-asset
  # .sha256 sidecars, so there is no portable fallback to swap in -- if a future
  # pinned version ever lacks .digest, this fails closed (the safe outcome) and
  # the version pin in .github/env should move to a release that carries one.
  expected_digest="$(gh api "repos/protocolbuffers/protobuf/releases/tags/v${version}" \
    --jq ".assets[] | select(.name == \"${asset}\") | .digest")"
  if [[ ! "$expected_digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
    echo "error: no usable sha256 digest for ${asset} (got: '${expected_digest:-<none>}')" >&2
    exit 1
  fi
  actual_sha="$(sha256_of "${tmp}/${asset}")"
  if [[ "$actual_sha" != "${expected_digest#sha256:}" ]]; then
    echo "error: checksum mismatch for ${asset}" >&2
    echo "  expected ${expected_digest#sha256:}" >&2
    echo "  actual   ${actual_sha}" >&2
    exit 1
  fi
  echo "Verified ${asset} sha256 ${actual_sha}"

  rm -rf "$PROTOC_HOME"
  mkdir -p "$PROTOC_HOME"
  unzip -q "${tmp}/${asset}" -d "$PROTOC_HOME"
  chmod +x "$protoc_bin"
fi

# Expose protoc on PATH for the remaining workflow steps.
if [[ -n "${GITHUB_PATH:-}" ]]; then
  echo "${PROTOC_HOME}/bin" >> "$GITHUB_PATH"
fi

"$protoc_bin" --version
