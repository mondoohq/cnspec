# Compliant: datastore role granted to a specific service account.
resource "google_project_iam_member" "datastore_user" {
  project = "my-project"
  role    = "roles/datastore.user"
  member  = "serviceAccount:app@my-project.iam.gserviceaccount.com"
}
