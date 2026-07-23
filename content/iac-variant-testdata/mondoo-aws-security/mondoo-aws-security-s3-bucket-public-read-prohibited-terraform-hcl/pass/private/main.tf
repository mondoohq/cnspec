resource "aws_s3_bucket_acl" "example" {
  bucket = "my-bucket"
  acl    = "private"
}
