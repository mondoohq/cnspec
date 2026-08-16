# Non-compliant: stage has X-Ray tracing disabled.
resource "aws_api_gateway_stage" "fail_example" {
  rest_api_id   = "abc123"
  stage_name    = "prod"
  deployment_id = "dep123"

  xray_tracing_enabled = false
}
