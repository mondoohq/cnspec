# Non-compliant: DynamoDB table encryption block lacks a customer-managed key ARN.
resource "aws_dynamodb_table" "fail_example" {
  name         = "fail-example"
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
