# Compliant: principal association uses a specific account, not a wildcard.
resource "aws_ram_principal_association" "pass_example" {
  principal          = "123456789012"
  resource_share_arn = "arn:aws:ram:us-east-1:123456789012:resource-share/example"
}
