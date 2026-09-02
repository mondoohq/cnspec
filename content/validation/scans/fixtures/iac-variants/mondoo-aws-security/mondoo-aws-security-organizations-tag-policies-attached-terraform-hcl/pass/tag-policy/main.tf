# Compliant: a tag policy is managed.
resource "aws_organizations_policy" "tags" {
  name    = "tag-standard"
  type    = "TAG_POLICY"
  content = "{}"
}
