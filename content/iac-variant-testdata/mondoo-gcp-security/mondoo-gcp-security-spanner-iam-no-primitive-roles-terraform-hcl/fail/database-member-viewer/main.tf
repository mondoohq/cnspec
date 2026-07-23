# Non-compliant: database IAM member is granted the primitive roles/viewer.
resource "google_spanner_database_iam_member" "fail_example" {
  instance = "my-instance"
  database = "my-database"
  role     = "roles/viewer"
  member   = "user:analyst@example.com"
}
