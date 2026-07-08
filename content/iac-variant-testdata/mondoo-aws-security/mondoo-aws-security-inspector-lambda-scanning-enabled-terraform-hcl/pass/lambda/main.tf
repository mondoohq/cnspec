# Compliant: Lambda scanning is enabled.
resource "aws_inspector2_enabler" "pass_example" {
  account_ids    = ["123456789012"]
  resource_types = ["LAMBDA"]
}
