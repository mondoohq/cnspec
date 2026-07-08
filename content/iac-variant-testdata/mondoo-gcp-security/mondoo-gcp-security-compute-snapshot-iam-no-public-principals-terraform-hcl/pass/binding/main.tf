# Compliant: IAM binding members are specific principals, not public.
resource "google_compute_snapshot_iam_binding" "pass_example" {
  project  = "my-project"
  snapshot = "my-snapshot"
  role     = "roles/compute.viewer"

  members = [
    "group:analysts@example.com",
    "serviceAccount:app@my-project.iam.gserviceaccount.com",
  ]
}
