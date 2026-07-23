# Non-compliant: IAM member grants access to allUsers (public).
resource "google_dataproc_cluster_iam_member" "fail_example" {
  project  = "my-project"
  region   = "us-central1"
  cluster  = "analytics-cluster"
  role     = "roles/dataproc.viewer"
  member   = "allUsers"
}
