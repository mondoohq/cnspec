# Non-compliant: instance IAM member is granted the primitive roles/owner.
resource "google_spanner_instance_iam_member" "fail_example" {
  instance = "my-instance"
  role     = "roles/owner"
  member   = "user:admin@example.com"
}
