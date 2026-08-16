# Non-compliant: one entry is scoped, but another opens the whole IPv4 range.
resource "aws_ec2_managed_prefix_list" "fail_mixed" {
  name           = "example"
  address_family = "IPv4"
  max_entries    = 5

  entry {
    cidr        = "10.0.0.0/16"
    description = "internal"
  }

  entry {
    cidr        = "0.0.0.0/0"
    description = "world"
  }
}
