# Compliant: IAM binding grants a predefined Dataproc role, not a primitive role.
resource "google_dataproc_cluster_iam_binding" "pass_example" {
  project  = "my-project"
  region   = "us-central1"
  cluster  = "analytics-cluster"
  role     = "roles/dataproc.viewer"

  members = [
    "group:data-team@example.com",
  ]
}
