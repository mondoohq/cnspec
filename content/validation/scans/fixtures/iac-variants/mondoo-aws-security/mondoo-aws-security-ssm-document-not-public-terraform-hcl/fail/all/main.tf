# Non-compliant: the permissions map shares the document publicly with "All".
resource "aws_ssm_document" "fail_example" {
  name            = "example-document"
  document_type   = "Command"
  document_format = "YAML"
  content         = "schemaVersion: '2.2'"

  permissions = {
    type        = "Share"
    account_ids = "All"
  }
}
