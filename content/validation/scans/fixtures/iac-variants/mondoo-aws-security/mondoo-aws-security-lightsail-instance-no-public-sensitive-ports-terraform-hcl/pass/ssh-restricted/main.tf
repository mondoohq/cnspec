# Compliant: SSH (22) is open only to a restricted CIDR, not 0.0.0.0/0.
resource "aws_lightsail_instance_public_ports" "pass_example" {
  instance_name = "example"

  port_info {
    protocol  = "tcp"
    from_port = 22
    to_port   = 22
    cidrs     = ["10.0.0.0/8"]
  }
}
