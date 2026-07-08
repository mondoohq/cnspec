# Compliant: Spanner database is encrypted with a customer-managed key.
resource "google_spanner_database" "pass_example" {
  instance = "my-instance"
  name     = "my-database"

  encryption_config {
    kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
  }
}
