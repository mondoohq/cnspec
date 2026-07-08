# Non-compliant: principal association uses a wildcard principal.
resource "aws_ram_principal_association" "fail_example" {
  principal          = "*"
  resource_share_arn = "arn:aws:ram:us-east-1:123456789012:resource-share/example"
}
