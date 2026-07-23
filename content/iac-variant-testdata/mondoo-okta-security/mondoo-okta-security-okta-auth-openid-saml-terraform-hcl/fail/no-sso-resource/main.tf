provider "okta" {
  org_name  = "example"
  base_url  = "okta.com"
  api_token = var.okta_api_token
}

resource "okta_user" "example" {
  first_name = "Jane"
  last_name  = "Doe"
  login      = "jane.doe@example.com"
  email      = "jane.doe@example.com"
}