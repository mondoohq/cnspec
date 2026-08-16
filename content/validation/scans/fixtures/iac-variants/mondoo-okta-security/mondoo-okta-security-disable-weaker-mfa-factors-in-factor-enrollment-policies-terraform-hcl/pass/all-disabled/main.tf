provider "okta" {
  org_name  = "example"
  base_url  = "okta.com"
  api_token = var.okta_api_token
}

resource "okta_policy_mfa_default" "example" {
  okta_sms = {
    enroll = "NOT_ALLOWED"
  }
  okta_email = {
    enroll = "NOT_ALLOWED"
  }
  phone_number = {
    enroll = "NOT_ALLOWED"
  }
}