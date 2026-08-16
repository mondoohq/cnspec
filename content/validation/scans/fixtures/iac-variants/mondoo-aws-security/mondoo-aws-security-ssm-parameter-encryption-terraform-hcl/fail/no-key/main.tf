# Non-compliant: SecureString but no explicit KMS key_id set.
resource "aws_ssm_parameter" "no_key" {
  name  = "/prod/db/password"
  type  = "SecureString"
  value = "s3cr3t"
}
