# Compliant: snapshot retention set to 0 (no snapshots to protect).
resource "aws_elasticache_replication_group" "pass_example" {
  replication_group_id     = "pass-example"
  description              = "pass example"
  snapshot_retention_limit = 0
}
