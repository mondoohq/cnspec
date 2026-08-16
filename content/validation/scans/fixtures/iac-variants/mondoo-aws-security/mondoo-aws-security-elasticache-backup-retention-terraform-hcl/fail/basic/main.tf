# Non-compliant: snapshot_retention_limit is 0 (no backups retained).
resource "aws_elasticache_replication_group" "fail_example" {
  replication_group_id     = "fail-example"
  description              = "fail example"
  snapshot_retention_limit = 0
}
