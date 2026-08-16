# Read-write, but only from the VPC range that hosts the application tier.
resource "alicloud_nas_access_rule" "app_tier" {
  access_group_name = alicloud_nas_access_group.shared.access_group_name
  source_cidr_ip    = "10.0.0.0/16"
  rw_access_type    = "RDWR"
  user_access_type  = "root_squash"
  priority          = 1
}
