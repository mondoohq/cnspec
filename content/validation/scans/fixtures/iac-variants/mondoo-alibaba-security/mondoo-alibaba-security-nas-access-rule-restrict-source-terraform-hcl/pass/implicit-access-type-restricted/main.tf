# No rw_access_type, so NAS applies RDWR, but only the application tier's
# range can mount.
resource "alicloud_nas_access_rule" "app_tier" {
  access_group_name = alicloud_nas_access_group.shared.access_group_name
  source_cidr_ip    = "10.0.0.0/16"
  priority          = 1
}
