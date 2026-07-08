# Non-compliant: IAM member grants access to allUsers (public).
resource "google_artifact_registry_repository_iam_member" "public" {
  location   = "us-central1"
  repository = "my-repo"
  role       = "roles/artifactregistry.reader"
  member     = "allUsers"
}
