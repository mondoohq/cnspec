# Non-compliant: organization policies are managed, but none opts out of AI services.
resource "aws_organizations_policy" "scp" {
  name    = "some-scp"
  content = "{}"
}
