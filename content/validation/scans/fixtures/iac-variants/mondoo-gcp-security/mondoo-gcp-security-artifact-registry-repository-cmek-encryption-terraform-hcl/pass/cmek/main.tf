# Compliant: repository encrypted with a customer-managed key (CMEK).
resource "google_artifact_registry_repository" "cmek" {
  location      = "us-central1"
  repository_id = "my-repo"
  format        = "DOCKER"
  kms_key_name  = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
}
