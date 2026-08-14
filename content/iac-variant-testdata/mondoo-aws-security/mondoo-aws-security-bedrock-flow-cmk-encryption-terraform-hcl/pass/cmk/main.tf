resource "aws_bedrockagent_flow" "enrichment" {
  name                        = "enrichment-flow"
  execution_role_arn          = aws_iam_role.bedrock_flow.arn
  customer_encryption_key_arn = aws_kms_key.bedrock.arn
  description                 = "Enriches inbound tickets before routing"
}
