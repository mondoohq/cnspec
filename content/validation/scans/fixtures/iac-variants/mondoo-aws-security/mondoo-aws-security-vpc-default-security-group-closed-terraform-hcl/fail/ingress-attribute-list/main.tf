resource "aws_default_security_group" "default" {
  vpc_id = aws_vpc.main.id

  ingress = [{
    protocol         = "tcp"
    from_port        = 22
    to_port          = 22
    cidr_blocks      = ["10.0.0.0/8"]
    description      = "ssh"
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    security_groups  = []
    self             = false
  }]
}
