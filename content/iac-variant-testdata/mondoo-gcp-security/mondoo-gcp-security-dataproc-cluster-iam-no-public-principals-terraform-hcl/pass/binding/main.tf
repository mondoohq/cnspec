# Compliant: IAM binding members are specific principals, not public.
resource "google_dataproc_cluster_iam_binding" "pass_example" {
  project  = "my-project"
  region   = "us-central1"
  cluster  = "analytics-cluster"
  role     = "roles/dataproc.viewer"

  members = [
    "group:analysts@example.com",
    "serviceAccount:app@my-project.iam.gserviceaccount.com",
  ]
}
