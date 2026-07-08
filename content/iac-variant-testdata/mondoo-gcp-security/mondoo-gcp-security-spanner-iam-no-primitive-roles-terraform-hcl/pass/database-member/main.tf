# Compliant: database IAM member uses a predefined non-primitive role.
resource "google_spanner_database_iam_member" "pass_example" {
  instance = "my-instance"
  database = "my-database"
  role     = "roles/spanner.databaseUser"
  member   = "serviceAccount:app@my-project.iam.gserviceaccount.com"
}
