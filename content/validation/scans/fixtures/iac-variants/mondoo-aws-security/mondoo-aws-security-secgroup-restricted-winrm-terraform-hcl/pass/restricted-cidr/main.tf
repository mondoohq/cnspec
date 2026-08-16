resource "aws_security_group" "winrm" {
  name = "winrm"

  ingress {
    from_port   = 5985
    to_port     = 5986
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]
  }
}
