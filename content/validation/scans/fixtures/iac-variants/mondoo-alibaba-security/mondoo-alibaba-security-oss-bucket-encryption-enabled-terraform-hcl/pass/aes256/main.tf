resource "alicloud_oss_bucket" "data" {
  bucket = "example-prod-data"
  acl    = "private"

  server_side_encryption_rule {
    sse_algorithm = "AES256"
  }
}
