# The mount target uses a custom permission group whose only rule is scoped
# to the application tier.
resource "alicloud_nas_access_group" "app" {
  access_group_name = "app-tier"
  access_group_type = "Vpc"
}

resource "alicloud_nas_access_rule" "app_tier" {
  access_group_name = alicloud_nas_access_group.app.access_group_name
  source_cidr_ip    = "10.0.0.0/16"
  rw_access_type    = "RDWR"
  user_access_type  = "root_squash"
  priority          = 1
}

resource "alicloud_nas_mount_target" "app" {
  file_system_id    = alicloud_nas_file_system.app.id
  access_group_name = alicloud_nas_access_group.app.access_group_name
  vswitch_id        = alicloud_vswitch.app.id
}
