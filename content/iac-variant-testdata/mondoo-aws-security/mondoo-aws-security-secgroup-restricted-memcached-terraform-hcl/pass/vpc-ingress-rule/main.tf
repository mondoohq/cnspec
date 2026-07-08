# Compliant: memcached (11211) restricted via aws_vpc_security_group_ingress_rule.
resource "aws_security_group" "memcached" {
  name = "memcached"
}

resource "aws_vpc_security_group_ingress_rule" "memcached" {
  security_group_id = aws_security_group.memcached.id
  from_port         = 11211
  to_port           = 11211
  ip_protocol       = "tcp"
  cidr_ipv4         = "10.0.0.0/16"
}
