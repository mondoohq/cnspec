resource "alicloud_oss_bucket_public_access_block" "data" {
  bucket              = alicloud_oss_bucket.data.bucket
  block_public_access = true
}
