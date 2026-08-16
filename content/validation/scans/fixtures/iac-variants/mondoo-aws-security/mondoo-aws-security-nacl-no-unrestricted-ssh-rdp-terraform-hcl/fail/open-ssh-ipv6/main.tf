# Non-compliant: ingress allow rule opens SSH (22) to all IPv6 addresses.
resource "aws_network_acl_rule" "fail_example" {
  network_acl_id  = "acl-12345678"
  rule_number     = 120
  egress          = false
  protocol        = "tcp"
  rule_action     = "allow"
  ipv6_cidr_block = "::/0"
  from_port       = 22
  to_port         = 22
}
