# serviceAccountUser at project scope allows attaching any service account to
# a new workload, inheriting its privileges.
resource "google_project_iam_binding" "sa_users" {
  project = var.project_id
  role    = "roles/iam.serviceAccountUser"

  members = [
    "group:developers@example.com",
  ]
}
