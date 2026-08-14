# Declaring the resource with the block turned off is worse than not declaring
# it: it reads as a deliberate decision to permit public access grants.
resource "alicloud_oss_bucket_public_access_block" "data" {
  bucket              = alicloud_oss_bucket.data.bucket
  block_public_access = false
}
