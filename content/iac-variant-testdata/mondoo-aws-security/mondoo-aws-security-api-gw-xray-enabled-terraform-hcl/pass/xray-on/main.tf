# Compliant: stage has X-Ray tracing enabled.
resource "aws_api_gateway_stage" "pass_example" {
  rest_api_id   = "abc123"
  stage_name    = "prod"
  deployment_id = "dep123"

  xray_tracing_enabled = true
}
