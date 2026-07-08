provider "okta" {
  org_name  = "example"
  base_url  = "okta.com"
  api_token = var.okta_api_token
}

resource "okta_security_notification_emails" "example" {
  report_suspicious_activity_enabled = false
}