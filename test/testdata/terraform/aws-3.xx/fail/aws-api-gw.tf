resource "aws_api_gateway_method_settings" "bad_example" {
  rest_api_id = aws_api_gateway_rest_api.example.id
  stage_name  = aws_api_gateway_stage.example.stage_name
  method_path = "path1/GET"
  settings {
    metrics_enabled = true
    logging_level   = "INFO"
    cache_data_encrypted = false
  }
}

resource "aws_apigatewayv2_stage" "fail_example" {
  api_id = aws_apigatewayv2_api.example.id
  name   = "example-stage"
}

resource "aws_api_gateway_stage" "fail_example" {
  stage_name    = "production"
  rest_api_id   = aws_api_gateway_rest_api.test.id
  deployment_id = aws_api_gateway_deployment.test.id
  xray_tracing_enabled = false
}

resource "aws_api_gateway_method" "fail_example" {
  rest_api_id   = aws_api_gateway_rest_api.SampleAPI.id
  resource_id   = aws_api_gateway_resource.SampleResource.id
  http_method   = "GET"
  authorization = "NONE"
}

resource "aws_api_gateway_domain_name" "fail_example" {
  security_policy = "TLS_1_0"
}