# Compliant: mysql (3306) ingress restricted to a private CIDR (block form).
resource "aws_security_group" "mysql" {
  name        = "mysql"
  description = "restricted mysql access"

  ingress {
    from_port   = 3306
    to_port     = 3306
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/16"]
  }
}
