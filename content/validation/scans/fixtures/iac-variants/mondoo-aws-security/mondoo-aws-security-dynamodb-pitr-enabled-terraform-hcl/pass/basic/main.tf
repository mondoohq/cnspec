# Compliant: point-in-time recovery enabled.
resource "aws_dynamodb_table" "pass_example" {
  name         = "pass-example"
  hash_key     = "id"
  billing_mode = "PAY_PER_REQUEST"

  attribute {
    name = "id"
    type = "S"
  }

  point_in_time_recovery {
    enabled = true
  }
}
