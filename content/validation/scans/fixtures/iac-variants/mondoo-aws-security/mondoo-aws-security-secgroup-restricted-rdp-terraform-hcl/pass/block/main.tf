# Compliant: RDP (3389) ingress restricted to a private CIDR (block form).
resource "aws_security_group" "rdp" {
  name        = "rdp"
  description = "restricted rdp access"

  ingress {
    from_port   = 3389
    to_port     = 3389
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/16"]
  }
}
