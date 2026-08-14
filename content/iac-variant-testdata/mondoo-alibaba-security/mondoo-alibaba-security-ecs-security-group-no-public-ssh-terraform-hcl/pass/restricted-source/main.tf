# SSH is reachable only from the corporate egress range.
resource "alicloud_security_group_rule" "ssh_from_office" {
  security_group_id = alicloud_security_group.app.id
  type              = "ingress"
  ip_protocol       = "tcp"
  port_range        = "22/22"
  cidr_ip           = "203.0.113.0/24"
  policy            = "accept"
  priority          = 1
}

# A world-open rule is fine here: 443 is the service's intended public port.
resource "alicloud_security_group_rule" "https_public" {
  security_group_id = alicloud_security_group.app.id
  type              = "ingress"
  ip_protocol       = "tcp"
  port_range        = "443/443"
  cidr_ip           = "0.0.0.0/0"
  policy            = "accept"
  priority          = 1
}
