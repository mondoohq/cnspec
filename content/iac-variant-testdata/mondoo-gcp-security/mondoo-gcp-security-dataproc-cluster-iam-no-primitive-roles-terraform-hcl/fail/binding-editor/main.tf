# Non-compliant: IAM binding grants the primitive roles/editor role.
resource "google_dataproc_cluster_iam_binding" "fail_example" {
  project  = "my-project"
  region   = "us-central1"
  cluster  = "analytics-cluster"
  role     = "roles/editor"

  members = [
    "group:developers@example.com",
  ]
}
