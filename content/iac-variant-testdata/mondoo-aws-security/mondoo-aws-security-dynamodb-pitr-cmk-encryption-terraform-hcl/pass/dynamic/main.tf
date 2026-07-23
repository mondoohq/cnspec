# Compliant: SSE-with-CMK and PITR both configured via conditional dynamic blocks.
variable "harden" {
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
    for_each = var.harden ? [1] : []
    content {
      enabled     = true
      kms_key_arn = "arn:aws:kms:us-east-1:111122223333:key/abcd"
    }
  }

  dynamic "point_in_time_recovery" {
    for_each = var.harden ? [1] : []
    content {
      enabled = true
    }
  }
}
