# An enabled integration with no allowed prefixes places no constraint on which
# endpoint an external function may be pointed at.
resource "snowflake_api_integration" "orders_gateway" {
  name                 = "ORDERS_GATEWAY"
  api_provider         = "aws_api_gateway"
  api_aws_role_arn     = "arn:aws:iam::123456789012:role/snowflake-api-gateway"
  api_allowed_prefixes = []
  enabled              = true
}
