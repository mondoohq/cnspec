# Non-compliant: one of two tables is missing point-in-time recovery.
resource "aws_dynamodb_table" "ok" {
  name         = "ok"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"
  attribute {
    name = "id"
    type = "S"
  }
  server_side_encryption {
    enabled     = true
    kms_key_arn = "arn:aws:kms:us-east-1:111122223333:key/abcd"
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
  server_side_encryption {
    enabled     = true
    kms_key_arn = "arn:aws:kms:us-east-1:111122223333:key/abcd"
  }
  point_in_time_recovery {
    enabled = false
  }
}
