# Non-compliant: folder log router settings do not set a CMEK key.
resource "google_logging_folder_settings" "fail_example" {
  folder               = "folders/987654321"
  disable_default_sink = false
}
