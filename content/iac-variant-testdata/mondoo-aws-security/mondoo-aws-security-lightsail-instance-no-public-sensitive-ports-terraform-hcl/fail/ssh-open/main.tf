# Non-compliant: SSH (22) open to the world.
resource "aws_lightsail_instance_public_ports" "fail_example" {
  instance_name = "example"

  port_info {
    protocol  = "tcp"
    from_port = 22
    to_port   = 22
    cidrs     = ["0.0.0.0/0"]
  }
}
