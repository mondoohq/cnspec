# Non-compliant: oracle (1521) open to the world via aws_vpc_security_group_ingress_rule.
resource "aws_security_group" "oracle" {
  name = "oracle"
}

resource "aws_vpc_security_group_ingress_rule" "oracle" {
  security_group_id = aws_security_group.oracle.id
  from_port         = 1521
  to_port           = 1521
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}
