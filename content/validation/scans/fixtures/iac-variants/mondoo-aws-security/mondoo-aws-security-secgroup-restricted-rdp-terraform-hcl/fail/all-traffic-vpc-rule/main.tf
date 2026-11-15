resource "aws_security_group" "example" {
  name = "all-traffic-open"
}

resource "aws_vpc_security_group_ingress_rule" "all" {
  security_group_id = aws_security_group.example.id
  ip_protocol       = "-1"
  cidr_ipv4         = "0.0.0.0/0"
}
