# Compliant: the document declares no permissions map, so it is not shared.
resource "aws_ssm_document" "pass_private" {
  name            = "private-document"
  document_type   = "Command"
  document_format = "YAML"
  content         = "schemaVersion: '2.2'"
}
