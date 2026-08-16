resource "alicloud_oss_bucket" "data" {
  bucket = "example-prod-data"
  acl    = "private"

  versioning {
    status = "Enabled"
  }
}
