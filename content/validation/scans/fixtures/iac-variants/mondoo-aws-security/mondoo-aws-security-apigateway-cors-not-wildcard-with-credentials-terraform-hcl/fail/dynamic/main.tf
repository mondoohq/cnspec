# Non-compliant: CORS block generated via a dynamic block still allows wildcard
# origins with credentials. Conditional dynamic blocks are a common Terraform
# idiom for optional configuration blocks.
variable "cors" {
  type = list(object({
    allow_credentials = bool
    allow_origins     = list(string)
  }))
  default = [{
    allow_credentials = true
    allow_origins     = ["*"]
  }]
}

resource "aws_apigatewayv2_api" "fail_dynamic" {
  name          = "example-api"
  protocol_type = "HTTP"

  dynamic "cors_configuration" {
    for_each = var.cors
    content {
      allow_credentials = cors_configuration.value.allow_credentials
      allow_origins     = cors_configuration.value.allow_origins
    }
  }
}
