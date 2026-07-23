provider "okta" {
  org_name  = "example"
  base_url  = "okta.com"
  api_token = var.okta_api_token
}

resource "okta_policy_mfa_default" "example" {
  okta_verify = {
    enroll = "NOT_ALLOWED"
  }
  okta_otp = {
    enroll = "NOT_ALLOWED"
  }
}