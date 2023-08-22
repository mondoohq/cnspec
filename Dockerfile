# Copyright (c) Mondoo, Inc.
# SPDX-License-Identifier: BUSL-1.1

ARG VERSION
FROM mondoo/cnquery:$VERSION
COPY cnspec /usr/local/bin
ENTRYPOINT ["cnspec"]
CMD ["help"]