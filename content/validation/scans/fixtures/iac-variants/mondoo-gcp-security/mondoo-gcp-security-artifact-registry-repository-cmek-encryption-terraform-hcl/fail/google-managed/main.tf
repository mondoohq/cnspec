# Non-compliant: no kms_key_name, so Google-managed encryption keys are used.
resource "google_artifact_registry_repository" "default_keys" {
  location      = "us-central1"
  repository_id = "my-repo"
  format        = "DOCKER"
}
