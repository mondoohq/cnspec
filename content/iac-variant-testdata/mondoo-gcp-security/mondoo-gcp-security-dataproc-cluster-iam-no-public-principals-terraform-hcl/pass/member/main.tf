# Compliant: IAM member is a specific principal, not public.
resource "google_dataproc_cluster_iam_member" "pass_example" {
  project  = "my-project"
  region   = "us-central1"
  cluster  = "analytics-cluster"
  role     = "roles/dataproc.viewer"
  member   = "user:alice@example.com"
}
