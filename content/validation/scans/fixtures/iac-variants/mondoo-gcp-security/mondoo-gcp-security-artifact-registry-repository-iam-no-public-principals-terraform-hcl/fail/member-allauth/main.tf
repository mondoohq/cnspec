# Non-compliant: IAM member grants access to allAuthenticatedUsers.
resource "google_artifact_registry_repository_iam_member" "public_auth" {
  location   = "us-central1"
  repository = "my-repo"
  role       = "roles/artifactregistry.reader"
  member     = "allAuthenticatedUsers"
}
