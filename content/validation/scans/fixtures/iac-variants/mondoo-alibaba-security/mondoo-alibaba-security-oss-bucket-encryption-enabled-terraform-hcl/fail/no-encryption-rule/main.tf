# No server_side_encryption_rule, so objects land unencrypted at rest.
resource "alicloud_oss_bucket" "data" {
  bucket = "example-prod-data"
  acl    = "private"
}
