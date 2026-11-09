# The Bedrock API key is valid until someone deletes it.
resource "aws_iam_service_specific_credential" "bedrock" {
  service_name = "bedrock.amazonaws.com"
  user_name    = "bedrock-dev"
}
