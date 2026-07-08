# Compliant: mssql (1433) restricted via aws_vpc_security_group_ingress_rule.
resource "aws_security_group" "mssql" {
  name = "mssql"
}

resource "aws_vpc_security_group_ingress_rule" "mssql" {
  security_group_id = aws_security_group.mssql.id
  from_port         = 1433
  to_port           = 1433
  ip_protocol       = "tcp"
  cidr_ipv4         = "10.0.0.0/16"
}
