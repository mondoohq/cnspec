# Compliant: mysql (3306) ingress restricted to a private CIDR (attribute-list form).
resource "aws_security_group" "mysql" {
  name        = "mysql"
  description = "restricted mysql access"

  ingress = [{
    from_port        = 3306
    to_port          = 3306
    protocol         = "tcp"
    cidr_blocks      = ["10.0.0.0/16"]
    description      = ""
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    security_groups  = []
    self             = false
  }]
}
