# Compliant: mssql (1433) ingress restricted to a private CIDR (attribute-list form).
resource "aws_security_group" "mssql" {
  name        = "mssql"
  description = "restricted mssql access"

  ingress = [{
    from_port        = 1433
    to_port          = 1433
    protocol         = "tcp"
    cidr_blocks      = ["10.0.0.0/16"]
    description      = ""
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    security_groups  = []
    self             = false
  }]
}
