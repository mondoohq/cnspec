# Non-compliant: oracle (1521) ingress open to the world (attribute-list form).
resource "aws_security_group" "oracle" {
  name        = "oracle"
  description = "open oracle access"

  ingress = [{
    from_port        = 1521
    to_port          = 1521
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
    description      = ""
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    security_groups  = []
    self             = false
  }]
}
