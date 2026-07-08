# Compliant: point-in-time recovery enabled via a conditional dynamic block.
variable "enable_pitr" {
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

  dynamic "point_in_time_recovery" {
    for_each = var.enable_pitr ? [1] : []
    content {
      enabled = true
    }
  }
}
