# Non-compliant: IAM binding grants access to allAuthenticatedUsers (public).
resource "google_dataproc_cluster_iam_binding" "fail_example" {
  project  = "my-project"
  region   = "us-central1"
  cluster  = "analytics-cluster"
  role     = "roles/dataproc.viewer"

  members = [
    "group:analysts@example.com",
    "allAuthenticatedUsers",
  ]
}
