resource "aws_secretsmanager_secret" "this" {
  name       = "example"
  kms_key_id = "arn:aws:kms:us-east-1:123456789012:key/abcd-1234"
}
