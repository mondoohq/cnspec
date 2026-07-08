# Compliant: oracle (1521) ingress restricted to a private CIDR (attribute-list form).
resource "aws_security_group" "oracle" {
  name        = "oracle"
  description = "restricted oracle access"

  ingress = [{
    from_port        = 1521
    to_port          = 1521
    protocol         = "tcp"
    cidr_blocks      = ["10.0.0.0/16"]
    description      = ""
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    security_groups  = []
    self             = false
  }]
}
