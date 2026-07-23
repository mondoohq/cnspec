# Non-compliant: GraphQL API uses API_KEY authentication only.
resource "aws_appsync_graphql_api" "fail_example" {
  name                = "example-api"
  authentication_type = "API_KEY"
}
