# Compliant: database IAM binding uses a predefined non-primitive role.
resource "google_spanner_database_iam_binding" "pass_example" {
  instance = "my-instance"
  database = "my-database"
  role     = "roles/spanner.databaseReader"

  members = [
    "group:analysts@example.com",
  ]
}
