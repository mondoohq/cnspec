resource "alicloud_ecs_disk" "data" {
  disk_name  = "prod-data"
  zone_id    = "cn-hangzhou-b"
  category   = "cloud_essd"
  size       = 500
  encrypted  = true
  kms_key_id = alicloud_kms_key.disk.id
}
