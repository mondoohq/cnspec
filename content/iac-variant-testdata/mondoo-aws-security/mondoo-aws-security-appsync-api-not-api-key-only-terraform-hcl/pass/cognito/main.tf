# Compliant: GraphQL API uses Cognito user pools, not API keys only.
resource "aws_appsync_graphql_api" "pass_example" {
  name                = "example-api"
  authentication_type = "AMAZON_COGNITO_USER_POOLS"

  user_pool_config {
    aws_region     = "us-east-1"
    default_action = "ALLOW"
    user_pool_id   = "us-east-1_example"
  }
}
