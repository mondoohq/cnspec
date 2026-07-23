# Compliant: HTTP and HTTPS are open to the world, while SSH is restricted to a
# private CIDR. No sensitive port is publicly exposed.
resource "aws_lightsail_instance_public_ports" "pass_example" {
  instance_name = "example"

  port_info {
    protocol  = "tcp"
    from_port = 80
    to_port   = 80
    cidrs     = ["0.0.0.0/0"]
  }

  port_info {
    protocol  = "tcp"
    from_port = 443
    to_port   = 443
    cidrs     = ["0.0.0.0/0"]
  }

  port_info {
    protocol  = "tcp"
    from_port = 22
    to_port   = 22
    cidrs     = ["10.0.0.0/8"]
  }
}
