provider "okta" {
  org_name  = "example"
  base_url  = "okta.com"
  api_token = var.okta_api_token
}

resource "okta_trusted_origin" "example" {
  name   = "Example"
  origin = "http://example.com"
  scopes = ["CORS", "REDIRECT"]
}
