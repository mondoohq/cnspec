# Compliant: restrictions block includes an application restriction.
resource "google_apikeys_key" "pass_example" {
  name         = "pass-key"
  display_name = "pass-key"

  restrictions {
    browser_key_restrictions {
      allowed_referrers = ["*.example.com"]
    }
  }
}
