# Non-compliant: datastore role granted to allUsers (public).
resource "google_project_iam_member" "datastore_public" {
  project = "my-project"
  role    = "roles/datastore.owner"
  member  = "allUsers"
}
