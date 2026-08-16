# Non-compliant: counted tables with SSE but no customer-managed KMS key.
resource "aws_dynamodb_table" "counted" {
  count        = 2
  name         = "counted-${count.index}"
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
