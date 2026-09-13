provider "okta" {
  org_name  = "example"
  base_url  = "okta.com"
  api_token = var.okta_api_token
}

resource "okta_policy_mfa_default" "example" {
  is_oie = true

  okta_verify = {
    enroll = "REQUIRED"
  }
  okta_email = {
    enroll = "NOT_ALLOWED"
  }
  phone_number = {
    enroll = "NOT_ALLOWED"
  }
}

resource "okta_policy_mfa" "contractors" {
  name   = "Contractors"
  is_oie = true

  okta_verify = {
    enroll = "REQUIRED"
  }
  okta_email = {
    enroll = "NOT_ALLOWED"
  }
  phone_number = {
    enroll = "NOT_ALLOWED"
  }
}
