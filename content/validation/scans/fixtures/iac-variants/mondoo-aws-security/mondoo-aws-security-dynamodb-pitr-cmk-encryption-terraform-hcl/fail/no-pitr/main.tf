# Non-compliant: point-in-time recovery disabled.
resource "aws_dynamodb_table" "fail_example" {
  name         = "fail-example"
  hash_key     = "id"
  billing_mode = "PAY_PER_REQUEST"

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
