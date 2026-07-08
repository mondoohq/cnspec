# Non-compliant: DynamoDB table has no server_side_encryption block.
resource "aws_dynamodb_table" "fail_example" {
  name         = "fail-example"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }
}
