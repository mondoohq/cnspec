# Non-compliant, written with the attribute-list form of ingress/egress.
# Terraform accepts both `ingress { ... }` (block) and `ingress = [{ ... }]`
# (attribute list); the check must catch both forms.
resource "aws_default_network_acl" "fail_example" {
  default_network_acl_id = "acl-12345678"

  ingress = [{
    protocol   = -1
    rule_no    = 100
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }]

  egress = [{
    protocol   = -1
    rule_no    = 100
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }]
}
