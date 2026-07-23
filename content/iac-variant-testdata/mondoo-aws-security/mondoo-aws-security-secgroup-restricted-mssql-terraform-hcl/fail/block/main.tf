# Non-compliant: mssql (1433) ingress open to the world (block form).
resource "aws_security_group" "mssql" {
  name        = "mssql"
  description = "open mssql access"

  ingress {
    from_port   = 1433
    to_port     = 1433
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
