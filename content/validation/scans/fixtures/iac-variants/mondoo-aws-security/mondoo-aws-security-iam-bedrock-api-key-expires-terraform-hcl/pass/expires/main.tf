# The Bedrock API key stops working after 30 days.
resource "aws_iam_service_specific_credential" "bedrock" {
  service_name        = "bedrock.amazonaws.com"
  user_name           = "bedrock-dev"
  credential_age_days = 30
}
