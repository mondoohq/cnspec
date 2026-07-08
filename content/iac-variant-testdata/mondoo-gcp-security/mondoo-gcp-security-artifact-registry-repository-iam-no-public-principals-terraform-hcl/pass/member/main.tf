# Compliant: IAM member grants access to a specific user, not a public principal.
resource "google_artifact_registry_repository_iam_member" "member" {
  location   = "us-central1"
  repository = "my-repo"
  role       = "roles/artifactregistry.reader"
  member     = "user:jane@example.com"
}
