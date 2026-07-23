# Non-compliant: document is shared publicly with "All".
resource "aws_ssm_document_permission" "fail_example" {
  document_name   = "example-document"
  permission_type = "Share"
  account_ids     = ["All"]
}
