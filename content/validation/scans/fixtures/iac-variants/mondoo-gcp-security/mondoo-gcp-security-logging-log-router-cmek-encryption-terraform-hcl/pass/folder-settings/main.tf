# Compliant: folder log router settings specify a CMEK key.
resource "google_logging_folder_settings" "pass_example" {
  folder       = "folders/987654321"
  kms_key_name = "projects/my-project/locations/global/keyRings/logs/cryptoKeys/router-key"
}
