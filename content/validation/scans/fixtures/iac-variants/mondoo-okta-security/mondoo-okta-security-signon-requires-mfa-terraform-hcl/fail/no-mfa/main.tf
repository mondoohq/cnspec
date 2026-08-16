provider "okta" {
  org_name  = "example"
  base_url  = "okta.com"
  api_token = var.okta_api_token
}

resource "okta_policy_rule_signon" "example" {
  name         = "Allow"
  mfa_required = false
}
