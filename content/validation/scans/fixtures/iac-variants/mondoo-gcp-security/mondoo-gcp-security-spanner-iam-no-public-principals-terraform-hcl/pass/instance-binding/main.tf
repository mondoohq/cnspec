# Compliant: instance IAM binding members are specific principals, not public.
resource "google_spanner_instance_iam_binding" "pass_example" {
  instance = "my-instance"
  role     = "roles/spanner.viewer"

  members = [
    "group:db-ops@example.com",
    "serviceAccount:app@my-project.iam.gserviceaccount.com",
  ]
}
