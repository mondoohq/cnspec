# Non-compliant: the key has no restrictions block, so it is usable from anywhere against every enabled API.
resource "google_apikeys_key" "fail_example" {
  name         = "fail-key"
  display_name = "fail-key"
}
