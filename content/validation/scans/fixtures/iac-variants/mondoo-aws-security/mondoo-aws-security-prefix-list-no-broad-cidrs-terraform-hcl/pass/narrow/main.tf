# Compliant: prefix list entries use narrow CIDRs.
resource "aws_ec2_managed_prefix_list" "pass_example" {
  name           = "example"
  address_family = "IPv4"
  max_entries    = 5

  entry {
    cidr        = "10.0.0.0/16"
    description = "internal"
  }
}
