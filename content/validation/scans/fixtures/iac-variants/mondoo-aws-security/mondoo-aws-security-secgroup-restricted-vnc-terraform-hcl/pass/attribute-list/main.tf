resource "aws_security_group" "vnc" {
  name = "vnc"

  ingress = [{
    description      = "VNC from internal network"
    from_port        = 5900
    to_port          = 5903
    protocol         = "tcp"
    cidr_blocks      = ["10.0.0.0/8"]
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    security_groups  = []
    self             = false
  }]
}
