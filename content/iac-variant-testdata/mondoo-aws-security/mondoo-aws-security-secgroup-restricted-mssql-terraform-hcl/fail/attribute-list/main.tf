# Non-compliant: mssql (1433) ingress open to the world (attribute-list form).
resource "aws_security_group" "mssql" {
  name        = "mssql"
  description = "open mssql access"

  ingress = [{
    from_port        = 1433
    to_port          = 1433
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
    description      = ""
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    security_groups  = []
    self             = false
  }]
}
