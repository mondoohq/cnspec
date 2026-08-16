resource "alicloud_instance" "app" {
  instance_name   = "prod-app-01"
  instance_type   = "ecs.g6.large"
  image_id        = "aliyun_3_x64_20G_alibase_20240819.vhd"
  security_groups = [alicloud_security_group.app.id]
  vswitch_id      = alicloud_vswitch.app.id

  http_endpoint = "enabled"
  http_tokens   = "required"
}
