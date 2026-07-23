# Non-compliant: IAM member grants the primitive roles/owner role.
resource "google_bigtable_instance_iam_member" "fail_example" {
  instance = "my-instance"
  role     = "roles/owner"
  member   = "user:alice@example.com"
}
