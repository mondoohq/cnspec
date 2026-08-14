# Disks are unencrypted unless the flag is set, so snapshots and the underlying
# storage hold plaintext.
resource "alicloud_ecs_disk" "data" {
  disk_name = "prod-data"
  zone_id   = "cn-hangzhou-b"
  category  = "cloud_essd"
  size      = 500
  encrypted = false
}
