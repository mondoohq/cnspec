variable "ssh_cidrs" {
  type    = list(string)
  default = ["0.0.0.0/0"]
}

resource "aws_security_group" "ex" {
  name        = "allow-ssh"
  description = "SSH from anywhere via a dynamic block"

  dynamic "ingress" {
    for_each = var.ssh_cidrs
    content {
      from_port   = 22
      to_port     = 22
      protocol    = "tcp"
      cidr_blocks = [ingress.value]
    }
  }
}
