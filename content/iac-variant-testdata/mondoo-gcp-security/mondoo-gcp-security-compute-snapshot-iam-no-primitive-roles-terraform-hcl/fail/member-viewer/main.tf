# Non-compliant: IAM member grants the primitive roles/viewer role.
resource "google_compute_snapshot_iam_member" "fail_example" {
  project  = "my-project"
  snapshot = "my-snapshot"
  role     = "roles/viewer"
  member   = "serviceAccount:reader@my-project.iam.gserviceaccount.com"
}
