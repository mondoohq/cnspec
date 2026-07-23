# Compliant: public port_info only opens HTTP (80), no sensitive ports.
resource "aws_lightsail_instance_public_ports" "pass_example" {
  instance_name = "example"

  port_info {
    protocol  = "tcp"
    from_port = 80
    to_port   = 80
    cidrs     = ["0.0.0.0/0"]
  }
}
