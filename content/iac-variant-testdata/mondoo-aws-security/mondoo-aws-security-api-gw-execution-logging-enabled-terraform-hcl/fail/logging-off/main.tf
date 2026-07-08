# Non-compliant: execution logging is turned OFF for all methods.
resource "aws_api_gateway_method_settings" "fail_example" {
  rest_api_id = "abc123"
  stage_name  = "prod"
  method_path = "*/*"

  settings {
    logging_level = "OFF"
  }
}
