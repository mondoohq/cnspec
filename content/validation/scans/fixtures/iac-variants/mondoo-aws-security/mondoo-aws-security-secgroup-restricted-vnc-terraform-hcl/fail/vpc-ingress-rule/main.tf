# Non-compliant: VNC (5900-5903) open to the world via the modern standalone
# aws_vpc_security_group_ingress_rule resource.
resource "aws_security_group" "vnc" {
  name = "vnc"
}

resource "aws_vpc_security_group_ingress_rule" "vnc" {
  security_group_id = aws_security_group.vnc.id
  from_port         = 5900
  to_port           = 5903
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}
