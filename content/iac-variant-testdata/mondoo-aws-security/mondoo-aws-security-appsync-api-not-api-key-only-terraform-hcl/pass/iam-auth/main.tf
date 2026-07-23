# Compliant: GraphQL API uses IAM authentication, not API keys only.
resource "aws_appsync_graphql_api" "pass_example" {
  name                = "example-api"
  authentication_type = "AWS_IAM"
}
