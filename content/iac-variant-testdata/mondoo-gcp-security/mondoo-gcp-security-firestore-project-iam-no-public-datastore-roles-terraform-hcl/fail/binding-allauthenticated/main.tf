# Non-compliant: datastore binding includes allAuthenticatedUsers (public).
resource "google_project_iam_binding" "datastore_binding" {
  project = "my-project"
  role    = "roles/datastore.user"
  members = [
    "serviceAccount:app@my-project.iam.gserviceaccount.com",
    "allAuthenticatedUsers",
  ]
}
