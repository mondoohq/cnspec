#!/usr/bin/env bash
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Resolves the cnspec binary the integration suite exercises.
#
# With VERSION set, downloads that release's archive and exports CNSPEC_BINARY,
# which test/integration honours in place of building from source. With VERSION
# empty, exports nothing and the suite builds the checked-out tree.
#
# The point is that a release candidate is tested as the artifact that will
# actually ship: packaging, ldflags and the production build tag included, none
# of which a build of the working tree exercises.
#
# The release archive, not install.mondoo.com: the download service resolves a
# channel, which is a second moving part between the input and the bytes. A
# release asset is addressed by tag and cannot drift.

set -euo pipefail

VERSION="${VERSION:-}"

if [ -z "${VERSION}" ]; then
  echo "no version requested: the suite will build cnspec from ${GITHUB_SHA:-the working tree}"
  exit 0
fi

# goreleaser's default archive name template:
#   {{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}
# .Version has no leading "v", the tag does. Verified against the published
# assets of v14.0.0-rc.10.
asset="cnspec_${VERSION#v}_linux_amd64.tar.gz"
dest="${RUNNER_TEMP:-/tmp}/cnspec-artifact"

# The release's direct download URL, not `gh release download`. The CLI finds
# the release through the REST lookup by tag, and GitHub can serve that lookup
# stale for a while after a release is published: for v14.0.0-rc.13 it reported
# 0 assets for over half an hour while the release by id, GraphQL and the
# download URLs all had 26. A download URL is addressed by tag and asset name
# and does not depend on that lookup.
#
# goreleaser's checksum file keeps the leading "v": cnspec_v14.0.0-rc.13_SHA256SUMS.
repo="${GITHUB_REPOSITORY:-mondoohq/cnspec}"
base="https://github.com/${repo}/releases/download/${VERSION}"
sums="cnspec_${VERSION}_SHA256SUMS"

mkdir -p "${dest}"
echo "downloading ${asset} from ${base}"
curl --fail --silent --show-error --location --retry 3 \
  --output "${dest}/${asset}" "${base}/${asset}"
curl --fail --silent --show-error --location --retry 3 \
  --output "${dest}/${sums}" "${base}/${sums}"

# Verify against the release's own checksums, so a truncated or substituted
# download fails here rather than as a confusing test failure.
(cd "${dest}" && grep " ${asset}\$" "${sums}" | sha256sum --check --strict)

tar -xzf "${dest}/${asset}" -C "${dest}"
chmod +x "${dest}/cnspec"

# Fail here rather than inside a scan if the archive did not contain what we
# expect, or is built for another architecture.
"${dest}/cnspec" version

echo "CNSPEC_BINARY=${dest}/cnspec" >>"${GITHUB_ENV}"
echo "testing artifact ${dest}/cnspec"
