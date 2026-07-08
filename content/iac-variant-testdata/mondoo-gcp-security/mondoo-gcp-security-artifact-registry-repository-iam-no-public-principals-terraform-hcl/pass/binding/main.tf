# Compliant: IAM binding grants access only to named service accounts and groups.
resource "google_artifact_registry_repository_iam_binding" "binding" {
  location   = "us-central1"
  repository = "my-repo"
  role       = "roles/artifactregistry.reader"
  members = [
    "serviceAccount:build@my-project.iam.gserviceaccount.com",
    "group:platform@example.com",
  ]
}
