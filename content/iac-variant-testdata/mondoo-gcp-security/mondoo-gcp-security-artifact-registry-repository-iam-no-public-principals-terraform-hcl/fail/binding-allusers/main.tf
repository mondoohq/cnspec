# Non-compliant: IAM binding includes allUsers among its members.
resource "google_artifact_registry_repository_iam_binding" "public" {
  location   = "us-central1"
  repository = "my-repo"
  role       = "roles/artifactregistry.reader"
  members = [
    "serviceAccount:build@my-project.iam.gserviceaccount.com",
    "allUsers",
  ]
}
