# Non-compliant: uses SSH connection requiring inbound access.
resource "aws_cloud9_environment_ec2" "example" {
  name            = "example"
  instance_type   = "t2.micro"
  connection_type = "CONNECT_SSH"
}
