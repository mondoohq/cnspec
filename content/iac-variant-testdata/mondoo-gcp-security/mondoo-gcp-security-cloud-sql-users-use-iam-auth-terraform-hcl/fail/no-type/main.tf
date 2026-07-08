# Non-compliant: no IAM type set, defaults to built-in password auth.
resource "google_sql_user" "app" {
  name     = "appuser"
  instance = google_sql_database_instance.main.name
  password = var.db_password
}
