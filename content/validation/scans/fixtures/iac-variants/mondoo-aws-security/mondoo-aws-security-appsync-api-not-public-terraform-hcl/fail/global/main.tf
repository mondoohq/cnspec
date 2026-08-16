# Non-compliant: GraphQL API visibility is GLOBAL (public).
resource "aws_appsync_graphql_api" "fail_example" {
  name                = "example-api"
  authentication_type = "AWS_IAM"
  visibility          = "GLOBAL"
}
