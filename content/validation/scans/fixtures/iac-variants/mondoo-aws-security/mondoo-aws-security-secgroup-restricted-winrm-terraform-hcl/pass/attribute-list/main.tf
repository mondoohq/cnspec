resource "aws_security_group" "winrm" {
  name = "winrm"

  ingress = [{
    description      = "WinRM from internal network"
    from_port        = 5985
    to_port          = 5986
    protocol         = "tcp"
    cidr_blocks      = ["10.0.0.0/8"]
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    security_groups  = []
    self             = false
  }]
}
