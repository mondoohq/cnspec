# AES256 is service-managed encryption: the keys are not customer-controlled,
# so key rotation and revocation stay with the provider.
resource "alicloud_oss_bucket" "data" {
  bucket = "example-prod-data"
  acl    = "private"

  server_side_encryption_rule {
    sse_algorithm = "AES256"
  }
}
