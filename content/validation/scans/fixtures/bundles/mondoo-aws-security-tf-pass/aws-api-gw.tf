resource "aws_api_gateway_method_settings" "pass_example" {
  rest_api_id = aws_api_gateway_rest_api.example.id
  stage_name  = aws_api_gateway_stage.example.stage_name
  method_path = "path1/GET"
  settings {
    metrics_enabled = true
    logging_level   = "INFO"
    cache_data_encrypted = true
  }
}

resource "aws_api_gateway_stage" "pass_example" {
  deployment_id = aws_api_gateway_deployment.example.id
  rest_api_id   = aws_api_gateway_rest_api.example.id
  stage_name    = "production"
  access_log_settings {
    destination_arn = "arn:aws:logs:region:account-id:log-group:log-group-name"
    format          = ""
  }
  xray_tracing_enabled = true
}

resource "aws_apigatewayv2_stage" "pass_example" {
  api_id = aws_apigatewayv2_api.example.id
  name   = "production"
  access_log_settings {
    destination_arn = "arn:aws:logs:region:account-id:log-group:log-group-name"
    format          = ""
  }
}

resource "aws_api_gateway_method" "pass_example_1" {
  rest_api_id   = aws_api_gateway_rest_api.SampleAPI.id
  resource_id   = aws_api_gateway_resource.SampleResource.id
  http_method   = "GET"
  authorization = "AWS_IAM"
}

resource "aws_api_gateway_method" "pass_example_2" {
  rest_api_id      = aws_api_gateway_rest_api.SampleAPI.id
  resource_id      = aws_api_gateway_resource.SampleResource.id
  http_method      = "GET"
  authorization    = "NONE"
  api_key_required = true
}

resource "aws_api_gateway_method" "pass_example_3" {
  rest_api_id   = aws_api_gateway_rest_api.SampleAPI.id
  resource_id   = aws_api_gateway_resource.SampleResource.id
  http_method   = "OPTIONS"
  authorization = "NONE"
}

resource "aws_api_gateway_domain_name" "pass_example" {
  security_policy = "TLS_1_2"
}