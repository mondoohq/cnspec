# Compliant: database IAM binding members are specific principals, not public.
resource "google_spanner_database_iam_binding" "pass_example" {
  instance = "my-instance"
  database = "my-database"
  role     = "roles/spanner.databaseUser"

  members = [
    "group:analysts@example.com",
  ]
}
