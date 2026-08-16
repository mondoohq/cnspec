# Two tables; the second enables SSE without a customer-managed KMS key.
resource "aws_dynamodb_table" "compliant" {
  name         = "compliant"
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
}

resource "aws_dynamodb_table" "violating" {
  name         = "violating"
  hash_key     = "id"
  billing_mode = "PAY_PER_REQUEST"
  attribute {
    name = "id"
    type = "S"
  }
  server_side_encryption {
    enabled = true
  }
}
