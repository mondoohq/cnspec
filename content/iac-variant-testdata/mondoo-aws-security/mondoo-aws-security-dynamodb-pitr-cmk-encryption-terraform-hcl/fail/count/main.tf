# Non-compliant: a counted table has PITR disabled.
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
    enabled     = true
    kms_key_arn = "arn:aws:kms:us-east-1:111122223333:key/abcd"
  }
  point_in_time_recovery {
    enabled = false
  }
}
