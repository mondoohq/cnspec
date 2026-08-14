resource "snowflake_saml2_integration" "okta" {
  name               = "OKTA_SSO"
  saml2_issuer       = "http://www.okta.com/exampleissuer"
  saml2_sso_url      = "https://example.okta.com/app/snowflake/exk1234/sso/saml"
  saml2_provider     = "OKTA"
  saml2_x509_cert    = "MIIDpDCCAoygAwIBAgIGAV2ka+55MA0GCSqGSIb3DQEBCwUAMIGSMQswCQYDVQQGEwJVUzETMBEG"
  saml2_sign_request = true
  enabled            = true
}
