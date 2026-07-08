# Compliant: GraphQL API visibility is set to PRIVATE.
resource "aws_appsync_graphql_api" "pass_example" {
  name                = "example-api"
  authentication_type = "AWS_IAM"
  visibility          = "PRIVATE"
}
