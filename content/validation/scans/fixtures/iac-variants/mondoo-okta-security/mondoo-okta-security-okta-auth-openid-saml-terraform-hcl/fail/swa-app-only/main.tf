# Non-compliant: the org declares an application, but it is a SWA app, which
# replays a stored username and password rather than federating. The check
# wants at least one app or IdP using SAML, OIDC or OAuth.
provider "okta" {
  org_name  = "example"
  base_url  = "okta.com"
  api_token = var.okta_api_token
}

resource "okta_app_swa" "legacy" {
  label          = "Legacy Intranet"
  button_field   = "btn-login"
  password_field = "txtbox-password"
  username_field = "txtbox-username"
  url            = "https://intranet.example.com/login.html"
}
