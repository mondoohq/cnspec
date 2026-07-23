# Compliant: organization log router settings specify a CMEK key.
resource "google_logging_organization_settings" "pass_example" {
  organization = "123456789"
  kms_key_name = "projects/my-project/locations/global/keyRings/logs/cryptoKeys/router-key"
}
