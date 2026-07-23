# Non-compliant: mongodb (27017) ingress open to the world (attribute-list form).
resource "aws_security_group" "mongodb" {
  name        = "mongodb"
  description = "open mongodb access"

  ingress = [{
    from_port        = 27017
    to_port          = 27017
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
    description      = ""
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    security_groups  = []
    self             = false
  }]
}
