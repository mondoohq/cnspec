# Non-compliant: mongodb (27017) open to the world via aws_vpc_security_group_ingress_rule.
resource "aws_security_group" "mongodb" {
  name = "mongodb"
}

resource "aws_vpc_security_group_ingress_rule" "mongodb" {
  security_group_id = aws_security_group.mongodb.id
  from_port         = 27017
  to_port           = 27017
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}
