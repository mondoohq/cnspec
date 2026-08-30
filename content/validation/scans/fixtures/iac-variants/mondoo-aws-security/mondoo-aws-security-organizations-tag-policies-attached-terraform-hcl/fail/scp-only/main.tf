# Non-compliant: organization policies are managed, but no tag policy exists.
resource "aws_organizations_policy" "scp" {
  name    = "some-scp"
  content = "{}"
}
