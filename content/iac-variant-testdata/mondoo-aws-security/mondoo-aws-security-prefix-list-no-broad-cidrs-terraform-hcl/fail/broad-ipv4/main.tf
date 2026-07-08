# Non-compliant: prefix list entry allows the entire IPv4 range.
resource "aws_ec2_managed_prefix_list" "fail_example" {
  name           = "example"
  address_family = "IPv4"
  max_entries    = 5

  entry {
    cidr        = "0.0.0.0/0"
    description = "world"
  }
}
