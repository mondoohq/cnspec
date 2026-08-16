resource "aws_security_group" "ssh" {
  name = "ssh"

  ingress = [{
    description      = "SSH from internal network"
    from_port        = 22
    to_port          = 22
    protocol         = "tcp"
    cidr_blocks      = ["10.0.0.0/8"]
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    security_groups  = []
    self             = false
  }]
}
