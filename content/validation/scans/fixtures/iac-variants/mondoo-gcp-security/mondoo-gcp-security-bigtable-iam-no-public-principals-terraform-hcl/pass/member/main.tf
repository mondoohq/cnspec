# Compliant: IAM member is a specific user, not a public principal.
resource "google_bigtable_instance_iam_member" "pass_example" {
  instance = "my-instance"
  role     = "roles/bigtable.user"
  member   = "user:alice@example.com"
}
