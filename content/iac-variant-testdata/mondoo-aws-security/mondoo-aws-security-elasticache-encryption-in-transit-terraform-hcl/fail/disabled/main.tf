# Non-compliant: in-transit encryption disabled.
resource "aws_elasticache_replication_group" "fail_example" {
  replication_group_id       = "fail-example"
  description                = "fail example"
  transit_encryption_enabled = false
}
