# Compliant: multiple entries, all scoped to specific ranges.
resource "aws_ec2_managed_prefix_list" "pass_multi" {
  name           = "example"
  address_family = "IPv4"
  max_entries    = 5

  entry {
    cidr        = "10.0.0.0/16"
    description = "internal-a"
  }

  entry {
    cidr        = "192.168.0.0/24"
    description = "internal-b"
  }
}
