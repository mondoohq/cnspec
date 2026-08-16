provider "okta" {
  org_name  = "example"
  base_url  = "okta.com"
  api_token = var.okta_api_token
}

resource "okta_trusted_origin" "secure" {
  name   = "Secure"
  origin = "https://app.example.com"
  scopes = ["CORS"]
}

resource "okta_trusted_origin" "insecure" {
  name   = "Insecure"
  origin = "http://legacy.example.com"
  scopes = ["REDIRECT"]
}
