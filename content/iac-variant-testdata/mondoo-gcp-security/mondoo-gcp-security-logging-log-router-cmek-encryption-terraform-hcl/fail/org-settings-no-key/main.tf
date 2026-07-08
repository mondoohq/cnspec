# Non-compliant: organization log router settings do not set a CMEK key.
resource "google_logging_organization_settings" "fail_example" {
  organization     = "123456789"
  disable_default_sink = false
}
