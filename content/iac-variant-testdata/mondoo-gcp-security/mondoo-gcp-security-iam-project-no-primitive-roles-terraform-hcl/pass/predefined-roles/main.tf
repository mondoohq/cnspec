# Compliant: both bindings grant scoped predefined roles rather than primitive roles.
resource "google_project_iam_member" "pass_member" {
  project = "my-project"
  role    = "roles/compute.viewer"
  member  = "user:alice@example.com"
}

resource "google_project_iam_binding" "pass_binding" {
  project = "my-project"
  role    = "roles/storage.objectAdmin"

  members = [
    "group:platform@example.com",
  ]
}
