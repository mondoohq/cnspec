# Non-compliant: RDP (3389) open to the world.
resource "aws_lightsail_instance_public_ports" "fail_example" {
  instance_name = "example"

  port_info {
    protocol  = "tcp"
    from_port = 80
    to_port   = 80
    cidrs     = ["0.0.0.0/0"]
  }

  port_info {
    protocol  = "tcp"
    from_port = 3389
    to_port   = 3389
    cidrs     = ["0.0.0.0/0"]
  }
}
