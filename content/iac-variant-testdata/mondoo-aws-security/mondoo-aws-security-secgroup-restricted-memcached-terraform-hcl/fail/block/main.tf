# Non-compliant: memcached (11211) ingress open to the world (block form).
resource "aws_security_group" "memcached" {
  name        = "memcached"
  description = "open memcached access"

  ingress {
    from_port   = 11211
    to_port     = 11211
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
