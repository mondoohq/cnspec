# Non-compliant: a wide port range (0-65535) open to the world spans every
# sensitive port, including SSH, RDP, and the database ports.
resource "aws_lightsail_instance_public_ports" "fail_example" {
  instance_name = "example"

  port_info {
    protocol  = "tcp"
    from_port = 0
    to_port   = 65535
    cidrs     = ["0.0.0.0/0"]
  }
}
