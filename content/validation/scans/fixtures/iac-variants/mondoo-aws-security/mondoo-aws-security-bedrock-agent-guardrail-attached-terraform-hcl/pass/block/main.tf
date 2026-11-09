# The agent's prompts and responses pass through a guardrail.
resource "aws_bedrockagent_agent" "support" {
  agent_name              = "support-agent"
  agent_resource_role_arn = "arn:aws:iam::111122223333:role/bedrock-agent"
  foundation_model        = "anthropic.claude-3-5-sonnet-20241022-v2:0"

  guardrail_configuration {
    guardrail_identifier = "a1b2c3d4e5f6"
    guardrail_version    = "1"
  }
}
