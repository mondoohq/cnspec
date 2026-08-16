resource "alicloud_oss_bucket" "data" {
  bucket = "example-prod-data"
  acl    = "private"

  logging {
    target_bucket = alicloud_oss_bucket.access_logs.id
    target_prefix = "oss-access/prod-data/"
  }
}
