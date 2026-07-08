# Compliant: NETWORK connection type has no connection_properties at all.
resource "aws_glue_connection" "pass_example" {
  name            = "example-network-connection"
  connection_type = "NETWORK"

  physical_connection_requirements {
    availability_zone      = "us-east-1a"
    security_group_id_list = ["sg-0123456789abcdef0"]
    subnet_id              = "subnet-0123456789abcdef0"
  }
}
