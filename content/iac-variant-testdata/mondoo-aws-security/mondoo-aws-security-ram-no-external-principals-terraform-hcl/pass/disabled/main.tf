# Compliant: resource share does not allow external principals.
resource "aws_ram_resource_share" "pass_example" {
  name                      = "example-share"
  allow_external_principals = false
}
