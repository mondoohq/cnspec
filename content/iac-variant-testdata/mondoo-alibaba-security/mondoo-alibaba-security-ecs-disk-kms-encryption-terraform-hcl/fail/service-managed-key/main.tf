# Encrypted, but with the service-managed key: the customer cannot rotate or
# revoke it, and cannot audit its use.
resource "alicloud_ecs_disk" "data" {
  disk_name = "prod-data"
  zone_id   = "cn-hangzhou-b"
  category  = "cloud_essd"
  size      = 500
  encrypted = true
}
