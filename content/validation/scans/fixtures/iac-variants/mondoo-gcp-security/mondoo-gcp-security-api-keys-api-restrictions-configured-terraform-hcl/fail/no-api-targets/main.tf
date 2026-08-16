# Non-compliant: restrictions block present but has no api_targets.
resource "google_apikeys_key" "fail_example" {
  name         = "fail-key"
  display_name = "fail-key"

  restrictions {
    browser_key_restrictions {
      allowed_referrers = ["*.example.com"]
    }
  }
}
