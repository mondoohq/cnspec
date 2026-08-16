# Non-compliant: SSE enabled but no customer-managed KMS key.
resource "aws_dynamodb_table" "fail_example" {
  name         = "fail-example"
  hash_key     = "id"
  billing_mode = "PAY_PER_REQUEST"

  attribute {
    name = "id"
    type = "S"
  }

  server_side_encryption {
    enabled = true
  }
}
