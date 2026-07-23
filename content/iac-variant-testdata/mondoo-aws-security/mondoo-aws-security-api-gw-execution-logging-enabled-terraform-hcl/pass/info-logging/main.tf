# Compliant: execution logging is enabled at INFO level for all methods.
resource "aws_api_gateway_method_settings" "pass_example" {
  rest_api_id = "abc123"
  stage_name  = "prod"
  method_path = "*/*"

  settings {
    logging_level = "INFO"
  }
}
