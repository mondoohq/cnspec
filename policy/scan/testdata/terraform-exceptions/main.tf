# Copyright (c) Mondoo, Inc.
# SPDX-License-Identifier: BUSL-1.1

resource "aws_s3_bucket" "fixtures" {
  bucket = "test-fixtures"
}
