# Non-compliant: prefix list entry allows the entire IPv6 range.
resource "aws_ec2_managed_prefix_list" "fail_example" {
  name           = "example"
  address_family = "IPv6"
  max_entries    = 5

  entry {
    cidr        = "::/0"
    description = "world"
  }
}
