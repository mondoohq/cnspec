provider "okta" {
  org_name  = "example"
  base_url  = "okta.com"
  api_token = var.okta_api_token
}

resource "okta_policy_password" "example" {
  name                   = "Example Password Policy"
  password_min_lowercase = var.min_lowercase
}
