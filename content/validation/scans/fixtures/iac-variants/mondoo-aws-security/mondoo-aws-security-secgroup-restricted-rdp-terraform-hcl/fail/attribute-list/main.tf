# Non-compliant: RDP (3389) ingress open to the world (attribute-list form).
resource "aws_security_group" "rdp" {
  name        = "rdp"
  description = "open rdp access"

  ingress = [{
    from_port        = 3389
    to_port          = 3389
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
    description      = ""
    ipv6_cidr_blocks = []
    prefix_list_ids  = []
    security_groups  = []
    self             = false
  }]
}
