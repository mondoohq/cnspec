resource "aws_bedrockagent_agent" "support" {
  agent_name                  = "support-agent"
  agent_resource_role_arn     = aws_iam_role.bedrock_agent.arn
  foundation_model            = "anthropic.claude-3-5-sonnet-20241022-v2:0"
  customer_encryption_key_arn = aws_kms_key.bedrock.arn
  instruction                 = "Answer customer questions using only the approved knowledge base."
}
