# Copyright Mondoo, Inc. 2026
# SPDX-License-Identifier: BUSL-1.1

ARG VERSION
FROM mondoo/mql:$VERSION
COPY cnspec /usr/local/bin
ENTRYPOINT ["cnspec"]
CMD ["help"]
