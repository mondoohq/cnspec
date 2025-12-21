
resource "aws_kms_key" "pass_example" {
  description             = "Key is used to encrypt bucket objects"
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_s3_bucket" "pass_example" {
  bucket = "test_bucket"
}

resource "aws_s3_bucket_acl" "pass_example" {
  bucket = aws_s3_bucket.pass_example.id
  acl = "private"
}

resource "aws_s3_bucket_versioning" "pass_example" {
  bucket = aws_s3_bucket.pass_example.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_logging" "pass_example" {
  bucket = aws_s3_bucket.pass_example.id
  target_bucket = aws_s3_bucket.pass_example_log1.id
  target_prefix = "log-${aws_s3_bucket.pass_example.id}/"
}

resource "aws_s3_bucket_server_side_encryption_configuration" "pass_example" {
bucket = aws_s3_bucket.pass_example.id
rule {
  apply_server_side_encryption_by_default {
    kms_master_key_id = aws_kms_key.pass_example.arn
    sse_algorithm = "aws:kms"
  }
}
}

resource "aws_s3_bucket_public_access_block" "pass_example" {
  bucket = aws_s3_bucket.pass_example.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Setup the logging bucket
resource "aws_s3_bucket" "pass_example_log1" {
  bucket = "${var.website-bucket-name}-logbucket"
}

resource "aws_s3_bucket_acl" "pass_example_log1" {
  bucket = aws_s3_bucket.pass_example_log1.id
  acl = "private"
}

resource "aws_s3_bucket_versioning" "pass_example_log1" {
  bucket = aws_s3_bucket.pass_example_log1.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "pass_example_log1" {
bucket = aws_s3_bucket.pass_example_log1.id
rule {
  apply_server_side_encryption_by_default {
    kms_master_key_id = aws_kms_key.pass_example.arn
    sse_algorithm = "aws:kms"
  }
}
}

resource "aws_s3_bucket_logging" "pass_example_log1" {
  bucket = aws_s3_bucket.pass_example_log1.id
  target_bucket = aws_s3_bucket.pass_example_log2.id
  target_prefix = "log-${aws_s3_bucket.pass_example_log1.id}/"
}

resource "aws_s3_bucket_public_access_block" "pass_example_log1" {
  bucket = aws_s3_bucket.pass_example_log1.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket" "pass_example_log2" {
  bucket = "${var.website-bucket-name}-logbucket2"
}

resource "aws_s3_bucket_acl" "pass_example_log2" {
  bucket = aws_s3_bucket.pass_example_log2.id
  acl = "private"
}

resource "aws_s3_bucket_versioning" "pass_example_log2" {
  bucket = aws_s3_bucket.pass_example_log2.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "pass_example_log2" {
bucket = aws_s3_bucket.pass_example_log2.id
rule {
  apply_server_side_encryption_by_default {
    kms_master_key_id = aws_kms_key.pass_example.arn
    sse_algorithm = "aws:kms"
  }
}
}

resource "aws_s3_bucket_logging" "pass_example_log2" {
  bucket = aws_s3_bucket.pass_example_log2.id
  target_bucket = aws_s3_bucket.pass_example_log1.id
  target_prefix = "log-${aws_s3_bucket.pass_example_log2.id}/"
}

resource "aws_s3_bucket_public_access_block" "pass_example_log2" {
  bucket = aws_s3_bucket.pass_example_log2.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_public_access_block" "pass_example" {
  bucket = aws_s3_bucket.pass_example.id

  block_public_policy = true
  block_public_acls = true
  ignore_public_acls = true
  restrict_public_buckets = true
}
