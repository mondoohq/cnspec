# Without a logging block there is no record of who read or wrote objects.
resource "alicloud_oss_bucket" "data" {
  bucket = "example-prod-data"
  acl    = "private"

  server_side_encryption_rule {
    sse_algorithm = "KMS"
  }
}
