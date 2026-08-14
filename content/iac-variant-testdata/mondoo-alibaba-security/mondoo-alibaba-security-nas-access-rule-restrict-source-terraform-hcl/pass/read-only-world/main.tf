# World-reachable but read-only, which the control permits.
resource "alicloud_nas_access_rule" "public_read" {
  access_group_name = alicloud_nas_access_group.shared.access_group_name
  source_cidr_ip    = "0.0.0.0/0"
  rw_access_type    = "RDONLY"
  user_access_type  = "no_squash"
  priority          = 1
}
