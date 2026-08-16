# Compliant: snapshots retained with at-rest encryption and a KMS key.
resource "aws_elasticache_replication_group" "pass_example" {
  replication_group_id       = "pass-example"
  description                = "pass example"
  snapshot_retention_limit   = 7
  at_rest_encryption_enabled = true
  kms_key_id                 = "arn:aws:kms:us-east-1:111122223333:key/abcd-1234"
}
