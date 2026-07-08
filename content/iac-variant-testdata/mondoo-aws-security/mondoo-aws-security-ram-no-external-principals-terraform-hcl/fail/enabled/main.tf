# Non-compliant: resource share allows external principals.
resource "aws_ram_resource_share" "fail_example" {
  name                      = "example-share"
  allow_external_principals = true
}
