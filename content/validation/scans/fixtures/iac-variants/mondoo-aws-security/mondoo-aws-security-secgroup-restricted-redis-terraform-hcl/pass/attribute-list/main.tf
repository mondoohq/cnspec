# Compliant: redis (6379) ingress restricted to a private CIDR (attribute-list form).
resource "aws_security_group" "redis" {
  name        = "redis"
  description = "restricted redis access"

  ingress = [{
    from_port        = 6379
    to_port          = 6379
    protocol         = "tcp"
    cidr_blocks      = ["10.0.0.0/16"]
    description      = ""
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    security_groups  = []
    self             = false
  }]
}
