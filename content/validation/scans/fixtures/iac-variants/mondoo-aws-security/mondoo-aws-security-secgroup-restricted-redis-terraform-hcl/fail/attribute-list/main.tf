# Non-compliant: redis (6379) ingress open to the world (attribute-list form).
resource "aws_security_group" "redis" {
  name        = "redis"
  description = "open redis access"

  ingress = [{
    from_port        = 6379
    to_port          = 6379
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
    description      = ""
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    security_groups  = []
    self             = false
  }]
}
