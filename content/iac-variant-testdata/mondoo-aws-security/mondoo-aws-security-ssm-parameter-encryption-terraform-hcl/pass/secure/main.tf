# Compliant: SecureString parameter encrypted with a customer-managed KMS key.
resource "aws_ssm_parameter" "secure" {
  name   = "/prod/db/password"
  type   = "SecureString"
  key_id = "alias/prod-ssm"
  value  = "s3cr3t"
}
