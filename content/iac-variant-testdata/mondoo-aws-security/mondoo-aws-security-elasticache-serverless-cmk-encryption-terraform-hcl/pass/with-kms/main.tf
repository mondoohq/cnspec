# Compliant: serverless cache uses a customer-managed KMS key.
resource "aws_elasticache_serverless_cache" "pass_example" {
  name       = "pass-example"
  engine     = "redis"
  kms_key_id = "arn:aws:kms:us-east-1:111122223333:key/abcd-1234"
}
