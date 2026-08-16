# Non-compliant: mysql (3306) ingress open to the world (block form).
resource "aws_security_group" "mysql" {
  name        = "mysql"
  description = "open mysql access"

  ingress {
    from_port   = 3306
    to_port     = 3306
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
