# Non-compliant: a counted table lacks a customer-managed KMS key.
resource "aws_dynamodb_table" "counted" {
  count        = 2
  name         = "example-${count.index}"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"
  attribute {
    name = "id"
    type = "S"
  }
  server_side_encryption {
    enabled = true
  }
}
