# Non-compliant: sensitive-port ingress via attribute-list form.
resource "aws_security_group" "example" {
  name = "example"

  ingress = [
    {
      description      = "app port"
      from_port        = 9200
      to_port          = 9300
      protocol         = "tcp"
      cidr_blocks      = ["0.0.0.0/0"]
      ipv6_cidr_blocks = ["fd00::/8"]
      prefix_list_ids  = []
      security_groups  = []
      self             = false
    }
  ]
}
