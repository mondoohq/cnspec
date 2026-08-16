# Non-compliant: a counted table has point-in-time recovery disabled.
resource "aws_dynamodb_table" "counted" {
  count        = 2
  name         = "example-${count.index}"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"
  attribute {
    name = "id"
    type = "S"
  }
  point_in_time_recovery {
    enabled = false
  }
}
