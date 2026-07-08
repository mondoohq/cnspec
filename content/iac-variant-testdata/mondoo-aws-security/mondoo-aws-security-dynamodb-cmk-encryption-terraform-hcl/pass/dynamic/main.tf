# Compliant: CMK server-side encryption configured via a conditional dynamic block.
variable "encrypt" {
  type    = bool
  default = true
}

resource "aws_dynamodb_table" "pass_dynamic" {
  name         = "pass-dynamic"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }

  dynamic "server_side_encryption" {
    for_each = var.encrypt ? [1] : []
    content {
      enabled     = true
      kms_key_arn = "arn:aws:kms:us-east-1:123456789012:key/abcd1234-a123-456a-a12b-a123b4cd56ef"
    }
  }
}
