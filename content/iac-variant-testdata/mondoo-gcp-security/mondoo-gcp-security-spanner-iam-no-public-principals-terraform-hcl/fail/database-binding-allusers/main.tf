# Non-compliant: database IAM binding grants access to allUsers (public).
resource "google_spanner_database_iam_binding" "fail_example" {
  instance = "my-instance"
  database = "my-database"
  role     = "roles/spanner.databaseReader"

  members = [
    "allUsers",
  ]
}
