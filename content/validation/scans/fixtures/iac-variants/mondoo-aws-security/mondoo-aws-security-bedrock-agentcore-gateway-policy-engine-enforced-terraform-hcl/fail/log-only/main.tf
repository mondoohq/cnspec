# Forbidden tool calls are logged but still go through.
resource "aws_bedrockagentcore_gateway" "example" {
  name            = "tools-gateway"
  role_arn        = "arn:aws:iam::111122223333:role/gateway"
  authorizer_type = "AWS_IAM"

  policy_engine_configuration {
    arn  = "arn:aws:bedrock-agentcore:us-east-1:111122223333:policy-engine/tools_engine-abcdef1234"
    mode = "LOG_ONLY"
  }
}
