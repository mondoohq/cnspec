# Non-compliant: egress via attribute-list form.
resource "aws_security_group" "example" {
  name = "example"

  egress = [
    {
      description      = "outbound"
      from_port        = 0
      to_port          = 0
      protocol         = "-1"
      cidr_blocks      = ["0.0.0.0/0"]
      ipv6_cidr_blocks = ["fd00::/8"]
      prefix_list_ids  = []
      security_groups  = []
      self             = false
    }
  ]
}
