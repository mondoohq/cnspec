# Compliant: IAM binding grants a predefined Bigtable role, not a primitive role.
resource "google_bigtable_instance_iam_binding" "pass_example" {
  instance = "my-instance"
  role     = "roles/bigtable.admin"

  members = [
    "group:bigtable-admins@example.com",
  ]
}
