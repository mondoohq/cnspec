# Non-compliant: at-rest encryption never configured (defaults off on older engines).
resource "aws_elasticache_replication_group" "fail_example" {
  replication_group_id = "fail-example"
  description          = "fail example"
  node_type           = "cache.t3.micro"
  num_cache_clusters  = 2
}
