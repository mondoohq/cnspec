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

mkdir -p "${dest}"
echo "downloading ${asset} from ${VERSION}"
gh release download "${VERSION}" \
  --repo "${GITHUB_REPOSITORY:-mondoohq/cnspec}" \
  --pattern "${asset}" \
  --dir "${dest}" \
  --clobber

tar -xzf "${dest}/${asset}" -C "${dest}"
chmod +x "${dest}/cnspec"

# Fail here rather than inside a scan if the archive did not contain what we
# expect, or is built for another architecture.
"${dest}/cnspec" version

echo "CNSPEC_BINARY=${dest}/cnspec" >>"${GITHUB_ENV}"
echo "testing artifact ${dest}/cnspec"
