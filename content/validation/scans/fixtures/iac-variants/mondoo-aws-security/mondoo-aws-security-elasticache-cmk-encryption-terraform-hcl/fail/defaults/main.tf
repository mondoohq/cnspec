# Non-compliant: at-rest encryption and KMS key both omitted (insecure defaults).
resource "aws_elasticache_replication_group" "fail_example" {
  replication_group_id = "fail-example"
  description          = "fail example"
}
