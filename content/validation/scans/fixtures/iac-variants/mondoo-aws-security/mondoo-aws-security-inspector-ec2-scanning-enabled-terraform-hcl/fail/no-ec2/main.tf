# Non-compliant: EC2 scanning is not enabled.
resource "aws_inspector2_enabler" "fail_example" {
  account_ids    = ["123456789012"]
  resource_types = ["ECR"]
}
