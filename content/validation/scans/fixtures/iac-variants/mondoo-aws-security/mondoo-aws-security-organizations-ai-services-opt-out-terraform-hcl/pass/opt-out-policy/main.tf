# Compliant: an AI services opt-out policy is managed.
resource "aws_organizations_policy" "ai_opt_out" {
  name    = "ai-services-opt-out"
  type    = "AISERVICES_OPT_OUT_POLICY"
  content = "{}"
}
