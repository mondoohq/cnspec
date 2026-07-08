# Non-compliant: database IAM member grants access to allAuthenticatedUsers.
resource "google_spanner_database_iam_member" "fail_example" {
  instance = "my-instance"
  database = "my-database"
  role     = "roles/spanner.databaseReader"
  member   = "allAuthenticatedUsers"
}
