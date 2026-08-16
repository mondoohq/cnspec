# Non-compliant: database IAM binding grants the primitive roles/owner.
resource "google_spanner_database_iam_binding" "fail_example" {
  instance = "my-instance"
  database = "my-database"
  role     = "roles/owner"

  members = [
    "group:db-ops@example.com",
  ]
}
