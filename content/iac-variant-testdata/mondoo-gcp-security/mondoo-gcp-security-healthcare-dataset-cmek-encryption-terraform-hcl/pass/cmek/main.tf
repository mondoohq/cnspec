# Compliant: healthcare dataset encrypted with a customer-managed KMS key.
resource "google_healthcare_dataset" "example" {
  name     = "example-dataset"
  location = "us-central1"

  encryption_spec {
    kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
  }
}
