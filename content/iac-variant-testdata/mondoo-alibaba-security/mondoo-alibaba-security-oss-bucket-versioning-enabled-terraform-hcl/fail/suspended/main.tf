# Suspended versioning stops retaining prior object versions, so an overwrite
# or delete is unrecoverable.
resource "alicloud_oss_bucket" "data" {
  bucket = "example-prod-data"
  acl    = "private"

  versioning {
    status = "Suspended"
  }
}
