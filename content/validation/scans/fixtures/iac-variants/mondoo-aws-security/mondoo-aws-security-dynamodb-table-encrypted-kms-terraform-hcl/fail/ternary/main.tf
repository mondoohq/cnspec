# Non-compliant: CMK toggled off by default, so kms_key_arn resolves to null
# (SSE falls back to an AWS-owned key). A common opt-in CMK pattern.
variable "use_cmk" {
  type    = bool
  default = false
}

resource "aws_dynamodb_table" "ternary" {
  name         = "ternary"
  hash_key     = "id"
  billing_mode = "PAY_PER_REQUEST"
  attribute {
    name = "id"
    type = "S"
  }
  server_side_encryption {
    enabled     = true
    kms_key_arn = var.use_cmk ? "arn:aws:kms:us-east-1:111122223333:key/aaaa" : null
  }
}
