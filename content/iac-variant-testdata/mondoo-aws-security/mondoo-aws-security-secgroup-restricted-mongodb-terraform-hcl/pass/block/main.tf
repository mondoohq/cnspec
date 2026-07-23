# Compliant: mongodb (27017) ingress restricted to a private CIDR (block form).
resource "aws_security_group" "mongodb" {
  name        = "mongodb"
  description = "restricted mongodb access"

  ingress {
    from_port   = 27017
    to_port     = 27017
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/16"]
  }
}
