# Compliant: uses SSM connection, no inbound SSH.
resource "aws_cloud9_environment_ec2" "example" {
  name            = "example"
  instance_type   = "t2.micro"
  connection_type = "CONNECT_SSM"
}
