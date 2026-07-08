# Non-compliant: connection_type omitted, defaulting to CONNECT_SSH which
# requires inbound access.
resource "aws_cloud9_environment_ec2" "example" {
  name          = "example"
  instance_type = "t2.micro"
}
