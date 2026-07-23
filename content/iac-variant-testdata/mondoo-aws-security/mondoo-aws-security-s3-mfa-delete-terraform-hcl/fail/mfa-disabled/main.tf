resource "aws_s3_bucket_versioning" "example" {
  bucket = "my-bucket"

  versioning_configuration {
    status     = "Enabled"
    mfa_delete = "Disabled"
  }
}
