# Compliant: document is shared only with specific account IDs.
resource "aws_ssm_document_permission" "pass_example" {
  document_name   = "example-document"
  permission_type = "Share"
  account_ids     = ["111122223333", "444455556666"]
}
