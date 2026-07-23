# Non-compliant: snapshots retained but without KMS encryption.
resource "aws_elasticache_replication_group" "fail_example" {
  replication_group_id       = "fail-example"
  description                = "fail example"
  snapshot_retention_limit   = 7
  at_rest_encryption_enabled = false
}
