# Compliant: database IAM member is a specific user, not public.
resource "google_spanner_database_iam_member" "pass_example" {
  instance = "my-instance"
  database = "my-database"
  role     = "roles/spanner.databaseReader"
  member   = "user:alice@example.com"
}
