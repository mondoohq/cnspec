# Non-compliant: plaintext String parameter, not encrypted at all.
resource "aws_ssm_parameter" "plaintext" {
  name  = "/prod/db/host"
  type  = "String"
  value = "db.example.internal"
}
