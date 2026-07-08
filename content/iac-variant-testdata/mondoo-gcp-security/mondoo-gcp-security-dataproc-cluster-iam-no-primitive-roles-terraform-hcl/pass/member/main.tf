# Compliant: IAM member grants a predefined Dataproc role, not a primitive role.
resource "google_dataproc_cluster_iam_member" "pass_example" {
  project  = "my-project"
  region   = "us-central1"
  cluster  = "analytics-cluster"
  role     = "roles/dataproc.editor"
  member   = "user:alice@example.com"
}
