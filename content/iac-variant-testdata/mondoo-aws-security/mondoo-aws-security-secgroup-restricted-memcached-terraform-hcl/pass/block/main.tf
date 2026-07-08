# Compliant: memcached (11211) ingress restricted to a private CIDR (block form).
resource "aws_security_group" "memcached" {
  name        = "memcached"
  description = "restricted memcached access"

  ingress {
    from_port   = 11211
    to_port     = 11211
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/16"]
  }
}
