# Non-compliant: ingress allow rule opens RDP (3389) to all IPv6 addresses.
resource "aws_network_acl_rule" "fail_example" {
  network_acl_id  = "acl-12345678"
  rule_number     = 130
  egress          = false
  protocol        = "tcp"
  rule_action     = "allow"
  ipv6_cidr_block = "::/0"
  from_port       = 3389
  to_port         = 3389
}
