# Non-compliant: restrictions block present but has no application restriction.
resource "google_apikeys_key" "fail_example" {
  name         = "fail-key"
  display_name = "fail-key"

  restrictions {
    api_targets {
      service = "translate.googleapis.com"
    }
  }
}
