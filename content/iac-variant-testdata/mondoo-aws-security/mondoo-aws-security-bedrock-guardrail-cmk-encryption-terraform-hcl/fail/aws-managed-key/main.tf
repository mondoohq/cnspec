# Guardrail configuration encodes the organization's content policy; without a
# customer key it is encrypted only with the AWS-managed one.
resource "aws_bedrock_guardrail" "content_policy" {
  name                      = "content-policy"
  blocked_input_messaging   = "This request cannot be processed."
  blocked_outputs_messaging = "This response cannot be returned."

  content_policy_config {
    filters_config {
      input_strength  = "HIGH"
      output_strength = "HIGH"
      type            = "HATE"
    }
  }
}
