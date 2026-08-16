# Compliant: at-rest encryption enabled.
resource "aws_elasticache_replication_group" "pass_example" {
  replication_group_id       = "pass-example"
  description                = "pass example"
  at_rest_encryption_enabled = true
}
