# Non-compliant: serverless cache without a KMS key.
resource "aws_elasticache_serverless_cache" "fail_example" {
  name   = "fail-example"
  engine = "redis"
}
