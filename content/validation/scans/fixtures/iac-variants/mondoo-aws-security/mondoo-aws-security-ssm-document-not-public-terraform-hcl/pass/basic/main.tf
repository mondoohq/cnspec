# Compliant: the document is shared only with named account IDs.
resource "aws_ssm_document" "pass_example" {
  name            = "example-document"
  document_type   = "Command"
  document_format = "YAML"
  content         = "schemaVersion: '2.2'"

  permissions = {
    type        = "Share"
    account_ids = "111122223333,444455556666"
  }
}
