# public-read exposes every object in the bucket to anonymous callers.
resource "alicloud_oss_bucket" "data" {
  bucket = "example-prod-data"
  acl    = "public-read"
}
