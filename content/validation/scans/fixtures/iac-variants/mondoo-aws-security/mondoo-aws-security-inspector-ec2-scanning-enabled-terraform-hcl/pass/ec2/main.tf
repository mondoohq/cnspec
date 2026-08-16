# Compliant: EC2 scanning is enabled.
resource "aws_inspector2_enabler" "pass_example" {
  account_ids    = ["123456789012"]
  resource_types = ["EC2"]
}
