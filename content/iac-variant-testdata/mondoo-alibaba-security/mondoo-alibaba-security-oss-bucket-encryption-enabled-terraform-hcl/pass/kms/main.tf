resource "alicloud_oss_bucket" "data" {
  bucket = "example-prod-data"
  acl    = "private"

  server_side_encryption_rule {
    sse_algorithm     = "KMS"
    kms_master_key_id = alicloud_kms_key.oss.id
  }
}
