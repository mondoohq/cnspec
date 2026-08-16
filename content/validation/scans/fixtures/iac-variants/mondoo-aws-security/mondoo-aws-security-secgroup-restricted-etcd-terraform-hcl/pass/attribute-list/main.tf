# Compliant: sensitive-port ingress via attribute-list form.
resource "aws_security_group" "example" {
  name = "example"

  ingress = [
    {
      description      = "app port"
      from_port        = 2379
      to_port          = 2380
      protocol         = "tcp"
      cidr_blocks      = ["10.0.0.0/8"]
      ipv6_cidr_blocks = ["fd00::/8"]
      prefix_list_ids  = []
      security_groups  = []
      self             = false
    }
  ]
}
