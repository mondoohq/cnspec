# Compliant: IAM member is a specific principal, not public.
resource "google_compute_snapshot_iam_member" "pass_example" {
  project  = "my-project"
  snapshot = "my-snapshot"
  role     = "roles/compute.viewer"
  member   = "user:alice@example.com"
}
