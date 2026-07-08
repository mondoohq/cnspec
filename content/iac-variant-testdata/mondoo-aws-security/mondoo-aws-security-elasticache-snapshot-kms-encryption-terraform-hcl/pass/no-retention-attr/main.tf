# Compliant: no snapshot_retention_limit set at all (no automatic snapshots).
resource "aws_elasticache_replication_group" "pass_example" {
  replication_group_id = "pass-example"
  description          = "pass example"
  node_type           = "cache.t3.micro"
  num_cache_clusters  = 2
}
