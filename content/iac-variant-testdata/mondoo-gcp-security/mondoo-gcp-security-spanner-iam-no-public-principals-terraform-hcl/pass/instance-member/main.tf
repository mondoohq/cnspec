# Compliant: instance IAM member is a specific service account, not public.
resource "google_spanner_instance_iam_member" "pass_example" {
  instance = "my-instance"
  role     = "roles/spanner.viewer"
  member   = "serviceAccount:app@my-project.iam.gserviceaccount.com"
}
