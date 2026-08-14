# Versioning is off by default when the block is omitted.
resource "alicloud_oss_bucket" "data" {
  bucket = "example-prod-data"
  acl    = "private"
}
