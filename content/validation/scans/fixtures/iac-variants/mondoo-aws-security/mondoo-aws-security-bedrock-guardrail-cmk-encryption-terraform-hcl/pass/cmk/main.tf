resource "aws_bedrock_guardrail" "content_policy" {
  name                      = "content-policy"
  blocked_input_messaging   = "This request cannot be processed."
  blocked_outputs_messaging = "This response cannot be returned."
  kms_key_arn               = aws_kms_key.bedrock.arn

  content_policy_config {
    filters_config {
      input_strength  = "HIGH"
      output_strength = "HIGH"
      type            = "HATE"
    }
  }
}
