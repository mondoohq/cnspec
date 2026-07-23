# Compliant: mysql (3306) restricted via aws_vpc_security_group_ingress_rule.
resource "aws_security_group" "mysql" {
  name = "mysql"
}

resource "aws_vpc_security_group_ingress_rule" "mysql" {
  security_group_id = aws_security_group.mysql.id
  from_port         = 3306
  to_port           = 3306
  ip_protocol       = "tcp"
  cidr_ipv4         = "10.0.0.0/16"
}
