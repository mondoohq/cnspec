# Non-compliant: an all-traffic (protocol -1) allow rule open to the world
# implicitly exposes SSH and RDP. The old check only matched from_port==22/3389.
resource "aws_network_acl_rule" "all_traffic" {
  network_acl_id = "acl-123"
  rule_number    = 100
  egress         = false
  protocol       = "-1"
  rule_action    = "allow"
  cidr_block     = "0.0.0.0/0"
}
