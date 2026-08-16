# Non-compliant: at-rest encryption enabled but no KMS key specified.
resource "aws_elasticache_replication_group" "fail_example" {
  replication_group_id      = "fail-example"
  description               = "fail example"
  at_rest_encryption_enabled = true
}
