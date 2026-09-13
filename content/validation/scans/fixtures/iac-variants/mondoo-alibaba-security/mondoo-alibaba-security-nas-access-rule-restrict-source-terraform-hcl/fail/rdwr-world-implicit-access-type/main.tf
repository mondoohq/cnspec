# No rw_access_type, so NAS applies its RDWR default and the rule is
# read-write from anywhere.
resource "alicloud_nas_access_rule" "open_default" {
  access_group_name = alicloud_nas_access_group.shared.access_group_name
  source_cidr_ip    = "0.0.0.0/0"
  priority          = 1
}
