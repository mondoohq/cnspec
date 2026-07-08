# Compliant: DynamoDB table uses a customer-managed KMS key for encryption.
resource "aws_dynamodb_table" "pass_example" {
  name         = "pass-example"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }

  server_side_encryption {
    enabled     = true
    kms_key_arn = "arn:aws:kms:us-east-1:123456789012:key/abcd1234-a123-456a-a12b-a123b4cd56ef"
  }
}
