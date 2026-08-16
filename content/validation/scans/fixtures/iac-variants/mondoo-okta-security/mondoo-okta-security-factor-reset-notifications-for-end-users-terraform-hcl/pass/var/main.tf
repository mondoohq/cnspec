provider "okta" {
  org_name  = "example"
  base_url  = "okta.com"
  api_token = var.okta_api_token
}

resource "okta_security_notification_emails" "example" {
  send_email_for_factor_reset_enabled = var.enable_send_email_for_factor_reset_enabled
}