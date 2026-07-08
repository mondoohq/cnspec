# Compliant: advanced security enforced via a conditional dynamic block.
variable "enable_addons" {
  type    = bool
  default = true
}

resource "aws_cognito_user_pool" "pass_dynamic" {
  name = "example"

  dynamic "user_pool_add_ons" {
    for_each = var.enable_addons ? [1] : []
    content {
      advanced_security_mode = "ENFORCED"
    }
  }
}
