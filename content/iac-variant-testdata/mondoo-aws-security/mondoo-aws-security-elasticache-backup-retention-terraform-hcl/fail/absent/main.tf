# Non-compliant: snapshot_retention_limit omitted, so backups default to disabled (0).
resource "aws_elasticache_replication_group" "fail_example" {
  replication_group_id = "fail-example"
  description          = "fail example"
}
