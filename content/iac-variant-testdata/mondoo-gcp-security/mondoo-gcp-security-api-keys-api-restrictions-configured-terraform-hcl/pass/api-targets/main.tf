# Compliant: restrictions block includes api_targets.
resource "google_apikeys_key" "pass_example" {
  name         = "pass-key"
  display_name = "pass-key"

  restrictions {
    api_targets {
      service = "translate.googleapis.com"
    }
  }
}
