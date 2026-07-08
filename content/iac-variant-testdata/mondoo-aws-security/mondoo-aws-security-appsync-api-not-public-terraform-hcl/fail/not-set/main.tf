# Non-compliant: visibility is not set, so it defaults to GLOBAL (public).
resource "aws_appsync_graphql_api" "fail_example" {
  name                = "example-api"
  authentication_type = "AWS_IAM"
}
