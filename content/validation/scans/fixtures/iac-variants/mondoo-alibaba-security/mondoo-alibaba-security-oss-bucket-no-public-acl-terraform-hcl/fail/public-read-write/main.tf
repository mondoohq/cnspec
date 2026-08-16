# public-read-write additionally lets anonymous callers upload and overwrite,
# which is how buckets get used to host malware.
resource "alicloud_oss_bucket" "uploads" {
  bucket = "example-prod-uploads"
  acl    = "public-read-write"
}
