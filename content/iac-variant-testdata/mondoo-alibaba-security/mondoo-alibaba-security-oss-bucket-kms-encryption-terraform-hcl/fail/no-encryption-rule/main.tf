# No encryption rule at all, so there is no customer-managed key either.
resource "alicloud_oss_bucket" "data" {
  bucket = "example-prod-data"
  acl    = "private"
}
