# Compliant: auth token set.
resource "aws_elasticache_replication_group" "pass_example" {
  replication_group_id       = "pass-example"
  description                = "pass example"
  transit_encryption_enabled = true
  auth_token                 = "aVeryLongAuthTokenValue1234567890"
}
