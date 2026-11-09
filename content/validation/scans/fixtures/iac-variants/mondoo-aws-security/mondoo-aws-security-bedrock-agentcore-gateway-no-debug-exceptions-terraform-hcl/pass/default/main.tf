# Callers receive generic error messages.
resource "aws_bedrockagentcore_gateway" "example" {
  name            = "tools-gateway"
  role_arn        = "arn:aws:iam::111122223333:role/gateway"
  authorizer_type = "AWS_IAM"
}
