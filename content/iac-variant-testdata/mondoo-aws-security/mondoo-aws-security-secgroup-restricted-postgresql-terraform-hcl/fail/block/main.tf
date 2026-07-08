# Non-compliant: postgresql (5432) ingress open to the world (block form).
resource "aws_security_group" "postgresql" {
  name        = "postgresql"
  description = "open postgresql access"

  ingress {
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
