# Non-compliant: no server_side_encryption block, so the table uses the default
# AWS-owned key rather than a customer-managed CMK.
resource "aws_dynamodb_table" "fail_example" {
  name         = "fail-example"
  hash_key     = "id"
  billing_mode = "PAY_PER_REQUEST"

  attribute {
    name = "id"
    type = "S"
  }
}
