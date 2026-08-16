# Compliant intent: SSE enabled and a customer-managed key selected via a
# ternary; both branches reference a non-empty KMS key ARN.
variable "primary_key" {
  type    = bool
  default = true
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
    kms_key_arn = var.primary_key ? "arn:aws:kms:us-east-1:111122223333:key/aaaa" : "arn:aws:kms:us-east-1:111122223333:key/bbbb"
  }
}
