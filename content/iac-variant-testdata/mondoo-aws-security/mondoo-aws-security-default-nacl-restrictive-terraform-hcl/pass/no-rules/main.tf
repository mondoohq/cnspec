# Compliant: default network ACL has no ingress or egress rules.
resource "aws_default_network_acl" "pass_example" {
  default_network_acl_id = "acl-12345678"
}
