provider "okta" {
  org_name  = "example"
  base_url  = "okta.com"
  api_token = var.okta_api_token
}

resource "okta_trusted_origin" "app" {
  name   = "App"
  origin = "https://app.example.com"
  scopes = ["CORS"]
}

resource "okta_trusted_origin" "portal" {
  name   = "Portal"
  origin = "https://portal.example.com"
  scopes = ["CORS", "REDIRECT"]
}
