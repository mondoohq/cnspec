# Agent state, including session data and prompts, is encrypted with the
# AWS-managed key rather than one the account controls.
resource "aws_bedrockagent_agent" "support" {
  agent_name              = "support-agent"
  agent_resource_role_arn = aws_iam_role.bedrock_agent.arn
  foundation_model        = "anthropic.claude-3-5-sonnet-20241022-v2:0"
  instruction             = "Answer customer questions using only the approved knowledge base."
}
