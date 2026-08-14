resource "snowflake_api_integration" "orders_gateway" {
  name                 = "ORDERS_GATEWAY"
  api_provider         = "aws_api_gateway"
  api_aws_role_arn     = "arn:aws:iam::123456789012:role/snowflake-api-gateway"
  api_allowed_prefixes = ["https://abc123.execute-api.us-east-1.amazonaws.com/prod/orders"]
  enabled              = true
}
