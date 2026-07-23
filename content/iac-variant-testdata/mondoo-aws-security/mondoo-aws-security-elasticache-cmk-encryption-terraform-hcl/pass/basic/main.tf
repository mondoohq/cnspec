# Compliant: at-rest encryption enabled with a customer-managed KMS key.
resource "aws_elasticache_replication_group" "pass_example" {
  replication_group_id      = "pass-example"
  description               = "pass example"
  at_rest_encryption_enabled = true
  kms_key_id                = var.kms_arn
}
