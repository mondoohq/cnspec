provider "okta" {
  org_name  = "example"
  base_url  = "okta.com"
  api_token = var.okta_api_token
}

resource "okta_policy_rule_signon" "example" {
  policy_id        = okta_policy_signon.example.id
  name             = "Example Rule"
  session_idle     = var.session_idle
  session_lifetime = var.session_lifetime
}