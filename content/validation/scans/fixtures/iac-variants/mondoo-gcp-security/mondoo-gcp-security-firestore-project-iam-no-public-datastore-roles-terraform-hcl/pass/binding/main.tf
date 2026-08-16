# Compliant: datastore binding restricted to named principals only.
resource "google_project_iam_binding" "datastore_viewer" {
  project = "my-project"
  role    = "roles/datastore.viewer"
  members = [
    "serviceAccount:reporting@my-project.iam.gserviceaccount.com",
    "group:data-team@example.com",
  ]
}
