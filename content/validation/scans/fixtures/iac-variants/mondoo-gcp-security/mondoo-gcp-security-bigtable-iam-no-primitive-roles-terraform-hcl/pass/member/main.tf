# Compliant: IAM member grants a predefined Bigtable role, not a primitive role.
resource "google_bigtable_instance_iam_member" "pass_example" {
  instance = "my-instance"
  role     = "roles/bigtable.user"
  member   = "user:alice@example.com"
}
