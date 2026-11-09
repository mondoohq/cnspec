# The agent is filtered only by the foundation model's own safeguards.
resource "aws_bedrockagent_agent" "support" {
  agent_name              = "support-agent"
  agent_resource_role_arn = "arn:aws:iam::111122223333:role/bedrock-agent"
  foundation_model        = "anthropic.claude-3-5-sonnet-20241022-v2:0"
}
