# Non-compliant: one of two tables has point-in-time recovery disabled.
resource "aws_dynamodb_table" "ok" {
  name         = "ok"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"
  attribute {
    name = "id"
    type = "S"
  }
  point_in_time_recovery {
    enabled = true
  }
}

resource "aws_dynamodb_table" "bad" {
  name         = "bad"
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
