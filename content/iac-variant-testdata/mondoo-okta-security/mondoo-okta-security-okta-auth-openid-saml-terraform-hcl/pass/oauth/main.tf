provider "okta" {
  org_name  = "example"
  base_url  = "okta.com"
  api_token = var.okta_api_token
}

resource "okta_app_oauth" "example" {
  label          = "Example OAuth App"
  type           = "web"
  grant_types    = ["authorization_code"]
  redirect_uris  = ["https://example.com/callback"]
  response_types = ["code"]
}