# Non-compliant: at-rest encryption disabled.
resource "aws_elasticache_replication_group" "fail_example" {
  replication_group_id       = "fail-example"
  description                = "fail example"
  at_rest_encryption_enabled = false
}
