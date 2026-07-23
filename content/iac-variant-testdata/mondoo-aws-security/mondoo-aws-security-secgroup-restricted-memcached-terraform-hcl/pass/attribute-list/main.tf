# Compliant: memcached (11211) ingress restricted to a private CIDR (attribute-list form).
resource "aws_security_group" "memcached" {
  name        = "memcached"
  description = "restricted memcached access"

  ingress = [{
    from_port        = 11211
    to_port          = 11211
    protocol         = "tcp"
    cidr_blocks      = ["10.0.0.0/16"]
    description      = ""
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    security_groups  = []
    self             = false
  }]
}
