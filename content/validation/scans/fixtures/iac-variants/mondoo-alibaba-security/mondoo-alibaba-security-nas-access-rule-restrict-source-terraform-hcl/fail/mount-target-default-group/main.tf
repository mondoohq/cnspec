# The default permission group grants read-write access to every address in
# the VPC and cannot be narrowed, so the mount target inherits that exposure.
resource "alicloud_nas_mount_target" "shared" {
  file_system_id    = alicloud_nas_file_system.shared.id
  access_group_name = "DEFAULT_VPC_GROUP_NAME"
  vswitch_id        = alicloud_vswitch.app.id
}
