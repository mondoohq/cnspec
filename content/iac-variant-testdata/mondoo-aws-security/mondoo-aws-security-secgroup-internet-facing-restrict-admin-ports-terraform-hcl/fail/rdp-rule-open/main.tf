resource "aws_security_group" "open" {
  name = "open"
}
resource "aws_vpc_security_group_ingress_rule" "rdp" {
  security_group_id = aws_security_group.open.id
  from_port         = 3389
  to_port           = 3389
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}
