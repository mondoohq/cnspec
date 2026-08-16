resource "aws_security_group" "vnc" {
  name = "vnc"

  ingress = [{
    description      = "VNC from anywhere"
    from_port        = 5900
    to_port          = 5903
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    security_groups  = []
    self             = false
  }]
}
