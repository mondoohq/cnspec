# Granted project-wide, serviceAccountTokenCreator lets the principal mint
# tokens for every service account in the project.
resource "google_project_iam_member" "token_creator" {
  project = var.project_id
  role    = "roles/iam.serviceAccountTokenCreator"
  member  = "group:developers@example.com"
}
