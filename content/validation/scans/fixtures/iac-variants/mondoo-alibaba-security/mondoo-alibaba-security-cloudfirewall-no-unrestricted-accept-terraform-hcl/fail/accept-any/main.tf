# Every protocol from every source to every destination. The account pays for
# Cloud Firewall and gets no enforcement from it.
resource "alicloud_cloud_firewall_control_policy" "allow_all" {
  direction        = "in"
  acl_action       = "accept"
  proto            = "ANY"
  source           = "0.0.0.0/0"
  source_type      = "net"
  destination      = "0.0.0.0/0"
  destination_type = "net"
  application_name = "ANY"
  description      = "temporary while debugging the deploy"
}
