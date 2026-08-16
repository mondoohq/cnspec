# The edge accepts one protocol on one port from one named range.
resource "alicloud_cloud_firewall_control_policy" "partner_api" {
  direction        = "in"
  acl_action       = "accept"
  proto            = "TCP"
  dest_port        = "443/443"
  dest_port_type   = "port"
  source           = "203.0.113.0/24"
  source_type      = "net"
  destination      = "10.0.0.0/16"
  destination_type = "net"
  application_name = "HTTPS"
  description      = "partner API callers"
}
