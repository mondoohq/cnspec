# Compliant: snapshot_retention_limit greater than 0.
resource "aws_elasticache_replication_group" "pass_example" {
  replication_group_id     = "pass-example"
  description              = "pass example"
  snapshot_retention_limit = 7
}
