# Non-compliant: IAM member grants the primitive roles/owner role.
resource "google_dataproc_cluster_iam_member" "fail_example" {
  project  = "my-project"
  region   = "us-central1"
  cluster  = "analytics-cluster"
  role     = "roles/owner"
  member   = "user:alice@example.com"
}
