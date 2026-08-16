# Read-write NFS export reachable from anywhere: anyone who can route to the
# mount target can overwrite the share.
resource "alicloud_nas_access_rule" "open_rw" {
  access_group_name = alicloud_nas_access_group.shared.access_group_name
  source_cidr_ip    = "0.0.0.0/0"
  rw_access_type    = "RDWR"
  user_access_type  = "no_squash"
  priority          = 1
}
