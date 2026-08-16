resource "aws_security_group" "winrm" {
  name   = "winrm"
  vpc_id = "vpc-12345678"
}

resource "aws_vpc_security_group_ingress_rule" "winrm" {
  security_group_id = aws_security_group.winrm.id
  from_port         = 5985
  to_port           = 5986
  ip_protocol       = "tcp"
  cidr_ipv4         = "10.0.0.0/8"
}
