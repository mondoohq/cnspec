# Compliant: postgresql (5432) ingress restricted to a private CIDR (block form).
resource "aws_security_group" "postgresql" {
  name        = "postgresql"
  description = "restricted postgresql access"

  ingress {
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/16"]
  }
}
