# A CodeCommit credential is not a Bedrock API key.
resource "aws_iam_service_specific_credential" "codecommit" {
  service_name = "codecommit.amazonaws.com"
  user_name    = "git-user"
}
